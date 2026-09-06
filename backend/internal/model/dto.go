package model

import (
	"errors"
	"time"
)

// APIResponse 统一响应结构
type APIResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// 统一错误码
const (
	CodeOK             = 0
	CodeParamError     = 40001
	CodeUnauthorized   = 40101
	CodeNotFound       = 40401
	CodeForbidden      = 40301
	CodeServerError    = 50001
	CodeWechatAPIError = 50002
)

// Pagination 分页
type Pagination struct {
	Page     int   `form:"page" json:"page"`
	PageSize int   `form:"page_size" json:"page_size"`
	Total    int64 `json:"total"`
}

func (p *Pagination) Normalize() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PageSize <= 0 || p.PageSize > 100 {
		p.PageSize = 20
	}
}

func (p *Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// ---- DTO ----

// LoginReq 登录请求
type LoginReq struct {
	Code string `json:"code" binding:"required"`
}

// LoginResp 登录响应
type LoginResp struct {
	Token   string `json:"token"`
	Expires int64  `json:"expires"` // Unix 时间戳
	User    *User  `json:"user"`
}

// UpdateUserReq 更新用户信息
type UpdateUserReq struct {
	Nickname  string `json:"nickname" binding:"required,max=64"`
	AvatarURL string `json:"avatar_url" binding:"max=512"`
}

// CreateFamilyReq 创建家庭
type CreateFamilyReq struct {
	Name      string `json:"name" binding:"required,max=64"`
	Role      string `json:"role" binding:"required,oneof=father mother grandparent other"`
	Nickname  string `json:"nickname" binding:"max=64"` // 家庭内称呼
	BabyName  string `json:"baby_name" binding:"required,max=64"`
	BabyGender int   `json:"baby_gender" binding:"oneof=0 1 2"`
	BabyBirthday *time.Time `json:"baby_birthday"`
}

// JoinFamilyReq 加入家庭
type JoinFamilyReq struct {
	InviteCode string `json:"invite_code" binding:"required,len=8"`
	Role       string `json:"role" binding:"required,oneof=father mother grandparent other"`
	Nickname   string `json:"nickname" binding:"max=64"`
}

// FamilyDetailResp 家庭详情（含成员和宝宝）
type FamilyDetailResp struct {
	Family    *Family         `json:"family"`
	Members   []*FamilyMember `json:"members"`
	Babies    []*Baby         `json:"babies"`
	MyRole    string          `json:"my_role"`
	MyNickname string         `json:"my_nickname"`
}

// CreateBabyReq 新增宝宝
type CreateBabyReq struct {
	Name     string      `json:"name" binding:"required,max=64"`
	Gender   int         `json:"gender" binding:"oneof=0 1 2"`
	Birthday *time.Time  `json:"birthday"`
}

// CareRecordReq 创建/更新护理记录
type CareRecordReq struct {
	BabyID    int64      `json:"baby_id" binding:"required"`
	Type      string     `json:"type" binding:"required,oneof=feeding sleep diaper temperature medicine"`
	StartedAt time.Time  `json:"started_at" binding:"required"`
	EndedAt   *time.Time `json:"ended_at"`
	AmountML  *float64   `json:"amount_ml" binding:"omitempty,gte=0,lte=2000"`
	Content   string     `json:"content" binding:"max=128"`
	TempValue *float64   `json:"temp_value" binding:"omitempty,gte=30,lte=45"`
	Note      string     `json:"note" binding:"max=512"`
	Details   MapJSON    `json:"details"`
}

// TodayStatsResp 今日看板统计
type TodayStatsResp struct {
	Date           string  `json:"date"`
	FeedingCount   int     `json:"feeding_count"`
	FeedingTotalML float64 `json:"feeding_total_ml"`
	SleepCount     int     `json:"sleep_count"`
	SleepTotalMin  int     `json:"sleep_total_min"`
	DiaperCount    int     `json:"diaper_count"`
	DiaperWetCount int     `json:"diaper_wet_count"`
	DiaperDirtyCount int   `json:"diaper_dirty_count"`
	LastFeedingAt  *time.Time `json:"last_feeding_at"`
	LastSleepEndAt *time.Time `json:"last_sleep_end_at"`
	LastDiaperAt   *time.Time `json:"last_diaper_at"`
	LatestTemp     *float64   `json:"latest_temp"`
	Records        []*CareRecord `json:"records"` // 今日全部记录（时间线）
}

// BusinessError 业务错误
type BusinessError struct {
	Code    int
	Message string
}

func (e *BusinessError) Error() string {
	return e.Message
}

func NewBizError(code int, msg string) *BusinessError {
	return &BusinessError{Code: code, Message: msg}
}

var ErrNotFound = errors.New("资源不存在")
var ErrForbidden = errors.New("无权访问该资源")
