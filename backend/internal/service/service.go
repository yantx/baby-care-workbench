package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"gorm.io/gorm"

	"github.com/yantx/baby-care-workbench/backend/internal/config"
	"github.com/yantx/baby-care-workbench/backend/internal/middleware"
	"github.com/yantx/baby-care-workbench/backend/internal/model"
	"github.com/yantx/baby-care-workbench/backend/internal/repository"
	"github.com/yantx/baby-care-workbench/backend/internal/ws"
)

// Service 业务逻辑层
type Service struct {
	Repo *repository.Repo
	Hub  *ws.Hub
}

func New(repo *repository.Repo, hub *ws.Hub) *Service {
	return &Service{Repo: repo, Hub: hub}
}

// ---------- 认证 ----------

// wechatSession 微信 code2Session 响应
type wechatSession struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

// Login 登录换取 JWT（Mock 模式 code 即设备标识；微信模式 code 换 openid）
func (s *Service) Login(ctx context.Context, code string, nickname string) (*model.LoginResp, error) {
	var (
		openid  string
		unionid string
	)

	if config.Get().Wechat.MockLogin() {
		// Mock 模式：本地开发无微信配置时，code 即设备标识，方便多端模拟不同家庭成员
		openid = "mock_" + code
		log.Printf("[auth] Mock登录 openid=%s", openid)
	} else {
		sess, err := s.code2Session(code)
		if err != nil {
			return nil, model.NewBizError(model.CodeWechatAPIError, "微信登录失败: "+err.Error())
		}
		openid, unionid = sess.OpenID, sess.UnionID
	}

	user, err := s.Repo.FindUserByOpenID(ctx, openid)
	if err != nil {
		return nil, err
	}
	if user == nil {
		nick := nickname
		if nick == "" {
			nick = "用户" + fmt.Sprintf("%06d", rand.Intn(1000000))
		}
		user = &model.User{
			OpenID:        openid,
			UnionID:       unionid,
			Nickname:      nick,
			ActiveFamilyID: 0,
		}
		if err := s.Repo.CreateUser(ctx, user); err != nil {
			return nil, err
		}
	}

	token, expires, err := middleware.IssueToken(user.ID, user.ActiveFamilyID)
	if err != nil {
		return nil, err
	}
	return &model.LoginResp{Token: token, Expires: expires, User: user}, nil
}

func (s *Service) code2Session(code string) (*wechatSession, error) {
	cfg := config.Get().Wechat
	api := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		url.QueryEscape(cfg.AppID), url.QueryEscape(cfg.Secret), url.QueryEscape(code),
	)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(api)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if err != nil {
		return nil, err
	}
	sess := &wechatSession{}
	if err := json.Unmarshal(body, sess); err != nil {
		return nil, err
	}
	if sess.ErrCode != 0 {
		return nil, fmt.Errorf("errcode=%d errmsg=%s", sess.ErrCode, sess.ErrMsg)
	}
	return sess, nil
}

// ---------- 用户 ----------

func (s *Service) UpdateProfile(ctx context.Context, userID int64, req *model.UpdateUserReq) (*model.User, error) {
	user, err := s.Repo.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	user.Nickname = req.Nickname
	user.AvatarURL = req.AvatarURL
	if err := s.Repo.UpdateUser(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// ---------- 家庭 ----------

// CreateFamily 创建家庭：同时写入成员（创建者）和宝宝档案
func (s *Service) CreateFamily(ctx context.Context, userID int64, req *model.CreateFamilyReq) (*model.FamilyDetailResp, error) {
	user, err := s.Repo.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	var family *model.Family
	err = s.Repo.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 生成唯一邀请码（6位大写字母+数字，冲突自动重试）
		inviteCode, err := genInviteCode(tx)
		if err != nil {
			return err
		}
		family = &model.Family{Name: req.Name, InviteCode: inviteCode, OwnerID: userID}
		if err := tx.Create(family).Error; err != nil {
			return err
		}
		member := &model.FamilyMember{
			FamilyID: family.ID, UserID: userID, Role: req.Role, Nickname: req.Nickname,
		}
		if err := tx.Create(member).Error; err != nil {
			return err
		}
		baby := &model.Baby{
			FamilyID: family.ID, Name: req.BabyName, Gender: req.BabyGender, Birthday: req.BabyBirthday,
		}
		if err := tx.Create(baby).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// 激活家庭
	user.ActiveFamilyID = family.ID
	_ = s.Repo.UpdateUser(ctx, user)

	return s.GetFamilyDetail(ctx, userID, family.ID)
}

// genInviteCode 生成唯一邀请码（去掉易混淆字符 0/O/1/I）
func genInviteCode(tx *gorm.DB) (string, error) {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	for i := 0; i < 10; i++ {
		code := make([]byte, 8)
		for j := range code {
			code[j] = chars[rand.Intn(len(chars))]
		}
		var count int64
		if err := tx.Model(&model.Family{}).Where("invite_code = ?", string(code)).Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return string(code), nil
		}
	}
	return "", fmt.Errorf("生成邀请码失败，请重试")
}

// JoinFamily 通过邀请码加入家庭
func (s *Service) JoinFamily(ctx context.Context, userID int64, req *model.JoinFamilyReq) (*model.FamilyDetailResp, error) {
	family, err := s.Repo.FindFamilyByInviteCode(ctx, req.InviteCode)
	if err != nil {
		return nil, err
	}
	if family == nil {
		return nil, model.NewBizError(model.CodeNotFound, "邀请码无效")
	}
	exist, err := s.Repo.GetMember(ctx, family.ID, userID)
	if err != nil {
		return nil, err
	}
	if exist == nil {
		member := &model.FamilyMember{
			FamilyID: family.ID, UserID: userID, Role: req.Role, Nickname: req.Nickname,
		}
		if err := s.Repo.CreateMember(ctx, member); err != nil {
			return nil, err
		}
		// 广播成员加入事件
		s.Hub.BroadcastToFamily(&ws.Message{
			Type: ws.MsgTypeMemberJoined, FamilyID: family.ID, SenderID: userID,
			Data: map[string]any{"user_id": userID, "role": req.Role},
		})
	}

	user, err := s.Repo.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	user.ActiveFamilyID = family.ID
	_ = s.Repo.UpdateUser(ctx, user)

	return s.GetFamilyDetail(ctx, userID, family.ID)
}

// GetFamilyDetail 家庭详情（校验成员身份）
func (s *Service) GetFamilyDetail(ctx context.Context, userID, familyID int64) (*model.FamilyDetailResp, error) {
	member, err := s.Repo.GetMember(ctx, familyID, userID)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, model.ErrForbidden
	}
	family, err := s.Repo.GetFamily(ctx, familyID)
	if err != nil {
		return nil, err
	}
	members, err := s.Repo.ListMembers(ctx, familyID)
	if err != nil {
		return nil, err
	}
	babies, err := s.Repo.ListBabies(ctx, familyID)
	if err != nil {
		return nil, err
	}
	babyResps := make([]*model.BabyResp, 0, len(babies))
	for _, b := range babies {
		babyResps = append(babyResps, toBabyResp(b))
	}
	return &model.FamilyDetailResp{
		Family: family, Members: members, Babies: babyResps,
		MyRole: member.Role, MyNickname: member.Nickname,
	}, nil
}

// toBabyResp 模型转响应（服务端计算出生天数，避免各端时区/解析差异）
func toBabyResp(b *model.Baby) *model.BabyResp {
	resp := &model.BabyResp{
		ID: b.ID, Name: b.Name, Gender: b.Gender, AvatarURL: b.AvatarURL,
	}
	if b.Birthday == nil {
		return resp
	}
	bd := b.Birthday.In(time.Local)
	resp.Birthday = bd.Format("2006-01-02")
	resp.DaysOld = calcDaysOld(bd)
	return resp
}

// calcDaysOld 按自然日计算出生第几天：出生当天=第1天
func calcDaysOld(birthday time.Time) int {
	loc := time.Local
	now := time.Now().In(loc)
	d0 := time.Date(birthday.Year(), birthday.Month(), birthday.Day(), 0, 0, 0, 0, loc)
	d1 := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	days := int(d1.Sub(d0).Hours()/24) + 1
	if days < 1 {
		days = 1
	}
	return days
}

// ---------- 宝宝 ----------

func (s *Service) CreateBaby(ctx context.Context, userID, familyID int64, req *model.CreateBabyReq) (*model.BabyResp, error) {
	member, err := s.Repo.GetMember(ctx, familyID, userID)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, model.ErrForbidden
	}
	baby := &model.Baby{FamilyID: familyID, Name: req.Name, Gender: req.Gender, Birthday: req.Birthday}
	if err := s.Repo.CreateBaby(ctx, baby); err != nil {
		return nil, err
	}
	return toBabyResp(baby), nil
}

// ---------- 护理记录 ----------

// CreateRecord 创建记录并广播
func (s *Service) CreateRecord(ctx context.Context, userID, familyID int64, req *model.CareRecordReq) (*model.CareRecord, error) {
	if _, err := s.assertMember(ctx, familyID, userID); err != nil {
		return nil, err
	}
	baby, err := s.Repo.GetBaby(ctx, req.BabyID)
	if err != nil {
		return nil, err
	}
	if baby.FamilyID != familyID {
		return nil, model.ErrForbidden
	}

	rec := &model.CareRecord{
		FamilyID: familyID, BabyID: req.BabyID, RecorderID: userID,
		Type: req.Type, StartedAt: req.StartedAt, EndedAt: req.EndedAt,
		AmountML: req.AmountML, Content: req.Content, TempValue: req.TempValue,
		Note: req.Note, Details: req.Details,
	}
	if err := s.Repo.CreateRecord(ctx, rec); err != nil {
		return nil, err
	}
	rec.Recorder, _ = s.Repo.GetUser(ctx, userID)

	s.Hub.BroadcastToFamily(&ws.Message{
		Type: ws.MsgTypeRecordCreated, FamilyID: familyID, SenderID: userID, Data: rec,
	})
	return rec, nil
}

// UpdateRecord 更新记录并广播
func (s *Service) UpdateRecord(ctx context.Context, userID, familyID, recordID int64, req *model.CareRecordReq) (*model.CareRecord, error) {
	if _, err := s.assertMember(ctx, familyID, userID); err != nil {
		return nil, err
	}
	rec, err := s.Repo.GetRecord(ctx, recordID)
	if err != nil {
		return nil, err
	}
	if rec.FamilyID != familyID {
		return nil, model.ErrForbidden
	}
	rec.StartedAt, rec.EndedAt = req.StartedAt, req.EndedAt
	rec.AmountML, rec.Content, rec.TempValue = req.AmountML, req.Content, req.TempValue
	rec.Note, rec.Details = req.Note, req.Details
	if err := s.Repo.UpdateRecord(ctx, rec); err != nil {
		return nil, err
	}
	s.Hub.BroadcastToFamily(&ws.Message{
		Type: ws.MsgTypeRecordUpdated, FamilyID: familyID, SenderID: userID, Data: rec,
	})
	return rec, nil
}

// DeleteRecord 删除记录并广播
func (s *Service) DeleteRecord(ctx context.Context, userID, familyID, recordID int64) error {
	if _, err := s.assertMember(ctx, familyID, userID); err != nil {
		return err
	}
	rec, err := s.Repo.GetRecord(ctx, recordID)
	if err != nil {
		return err
	}
	if rec.FamilyID != familyID {
		return model.ErrForbidden
	}
	if err := s.Repo.DeleteRecord(ctx, recordID); err != nil {
		return err
	}
	s.Hub.BroadcastToFamily(&ws.Message{
		Type: ws.MsgTypeRecordDeleted, FamilyID: familyID, SenderID: userID,
		Data: map[string]any{"id": recordID},
	})
	return nil
}

// ListRecords 分页查询记录
func (s *Service) ListRecords(ctx context.Context, userID, familyID int64, q repository.RecordQuery, page, size int) ([]*model.CareRecord, int64, error) {
	if _, err := s.assertMember(ctx, familyID, userID); err != nil {
		return nil, 0, err
	}
	q.FamilyID = familyID
	return s.Repo.ListRecords(ctx, q, page, size)
}

// TodayStats 今日看板：统计 + 当日时间线
func (s *Service) TodayStats(ctx context.Context, userID, familyID, babyID int64) (*model.TodayStatsResp, error) {
	if _, err := s.assertMember(ctx, familyID, userID); err != nil {
		return nil, err
	}
	loc := time.Local
	dayStart := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, loc)
	dayEnd := dayStart.Add(24 * time.Hour)

	q := repository.RecordQuery{FamilyID: familyID, BabyID: babyID, From: dayStart, To: dayEnd}
	records, _, err := s.Repo.ListRecords(ctx, q, 1, 500)
	if err != nil {
		return nil, err
	}

	resp := &model.TodayStatsResp{Date: dayStart.Format("2006-01-02"), Records: records}
	for _, r := range records {
		switch r.Type {
		case model.RecordTypeFeeding:
			resp.FeedingCount++
			if r.AmountML != nil {
				resp.FeedingTotalML += *r.AmountML
			}
			resp.LastFeedingAt = latestTime(resp.LastFeedingAt, &r.StartedAt)
		case model.RecordTypeSleep:
			resp.SleepCount++
			if r.EndedAt != nil {
				resp.SleepTotalMin += int(r.EndedAt.Sub(r.StartedAt).Minutes())
				resp.LastSleepEndAt = latestTime(resp.LastSleepEndAt, r.EndedAt)
			}
		case model.RecordTypeDiaper:
			resp.DiaperCount++
			if r.Content == "wet" {
				resp.DiaperWetCount++
			}
			if r.Content == "dirty" {
				resp.DiaperDirtyCount++
			}
			resp.LastDiaperAt = latestTime(resp.LastDiaperAt, &r.StartedAt)
		case model.RecordTypeTemperature:
			if r.TempValue != nil {
				resp.LatestTemp = r.TempValue // 记录按时间倒序，取到的即最新
			}
		}
	}
	return resp, nil
}

func latestTime(a, b *time.Time) *time.Time {
	if a == nil {
		return b
	}
	if b != nil && b.After(*a) {
		return b
	}
	return a
}

func (s *Service) assertMember(ctx context.Context, familyID, userID int64) (*model.FamilyMember, error) {
	member, err := s.Repo.GetMember(ctx, familyID, userID)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, model.ErrForbidden
	}
	return member, nil
}
