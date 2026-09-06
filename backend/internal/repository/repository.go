package repository

import (
	"context"
	"errors"
	"math/rand"

	"gorm.io/gorm"

	"github.com/yantx/baby-care-workbench/backend/internal/model"
)

// Repo 数据访问层（Phase 1 单库单实例，按模块拆方法；演进期可按聚合拆分为多 Repo）
type Repo struct {
	DB *gorm.DB
}

func New(db *gorm.DB) *Repo {
	return &Repo{DB: db}
}

// ---- User ----

func (r *Repo) FindUserByOpenID(ctx context.Context, openid string) (*model.User, error) {
	var u model.User
	err := r.DB.WithContext(ctx).Where("openid = ?", openid).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

func (r *Repo) CreateUser(ctx context.Context, u *model.User) error {
	return r.DB.WithContext(ctx).Create(u).Error
}

func (r *Repo) UpdateUser(ctx context.Context, u *model.User) error {
	return r.DB.WithContext(ctx).Model(u).
		Select("nickname", "avatar_url", "active_family_id").
		Updates(u).Error
}

func (r *Repo) GetUser(ctx context.Context, id int64) (*model.User, error) {
	var u model.User
	err := r.DB.WithContext(ctx).First(&u, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrNotFound
	}
	return &u, err
}

// ---- Family ----

// 生成 8 位不重复邀请码
func genInviteCode() string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func (r *Repo) CreateFamily(ctx context.Context, f *model.Family) error {
	// 邀请码冲突自动重试
	for i := 0; i < 3; i++ {
		f.InviteCode = genInviteCode()
		err := r.DB.WithContext(ctx).Create(f).Error
		if err == nil {
			return nil
		}
	}
	return r.DB.WithContext(ctx).Create(f).Error
}

func (r *Repo) GetFamily(ctx context.Context, id int64) (*model.Family, error) {
	var f model.Family
	err := r.DB.WithContext(ctx).First(&f, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrNotFound
	}
	return &f, err
}

func (r *Repo) FindFamilyByInviteCode(ctx context.Context, code string) (*model.Family, error) {
	var f model.Family
	err := r.DB.WithContext(ctx).Where("invite_code = ?", code).First(&f).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &f, err
}

func (r *Repo) ListMembers(ctx context.Context, familyID int64) ([]*model.FamilyMember, error) {
	var list []*model.FamilyMember
	err := r.DB.WithContext(ctx).
		Preload("User").
		Where("family_id = ?", familyID).
		Order("joined_at ASC").
		Find(&list).Error
	return list, err
}

func (r *Repo) GetMember(ctx context.Context, familyID, userID int64) (*model.FamilyMember, error) {
	var m model.FamilyMember
	err := r.DB.WithContext(ctx).
		Where("family_id = ? AND user_id = ?", familyID, userID).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &m, err
}

func (r *Repo) CreateMember(ctx context.Context, m *model.FamilyMember) error {
	return r.DB.WithContext(ctx).Create(m).Error
}

func (r *Repo) ListFamiliesByUser(ctx context.Context, userID int64) ([]*model.FamilyMember, error) {
	var list []*model.FamilyMember
	err := r.DB.WithContext(ctx).Preload("Family").Where("user_id = ?", userID).Find(&list).Error
	return list, err
}

// ---- Baby ----

func (r *Repo) CreateBaby(ctx context.Context, b *model.Baby) error {
	return r.DB.WithContext(ctx).Create(b).Error
}

func (r *Repo) ListBabies(ctx context.Context, familyID int64) ([]*model.Baby, error) {
	var list []*model.Baby
	err := r.DB.WithContext(ctx).Where("family_id = ?", familyID).Order("created_at ASC").Find(&list).Error
	return list, err
}

func (r *Repo) GetBaby(ctx context.Context, id int64) (*model.Baby, error) {
	var b model.Baby
	err := r.DB.WithContext(ctx).First(&b, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrNotFound
	}
	return &b, err
}

// ---- CareRecord ----

func (r *Repo) CreateRecord(ctx context.Context, rec *model.CareRecord) error {
	return r.DB.WithContext(ctx).Create(rec).Error
}

func (r *Repo) UpdateRecord(ctx context.Context, rec *model.CareRecord) error {
	return r.DB.WithContext(ctx).
		Model(rec).
		Select("started_at", "ended_at", "amount_ml", "content", "temp_value", "note", "details").
		Updates(rec).Error
}

func (r *Repo) DeleteRecord(ctx context.Context, id int64) error {
	return r.DB.WithContext(ctx).Delete(&model.CareRecord{}, id).Error
}

func (r *Repo) GetRecord(ctx context.Context, id int64) (*model.CareRecord, error) {
	var rec model.CareRecord
	err := r.DB.WithContext(ctx).First(&rec, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrNotFound
	}
	return &rec, err
}

type RecordQuery struct {
	FamilyID int64
	BabyID   int64
	Type     string
	From, To interface{} // 时间范围（含边界）
}

func (r *Repo) ListRecords(ctx context.Context, q RecordQuery, page, size int) ([]*model.CareRecord, int64, error) {
	db := r.DB.WithContext(ctx).Model(&model.CareRecord{}).Where("family_id = ?", q.FamilyID)
	if q.BabyID > 0 {
		db = db.Where("baby_id = ?", q.BabyID)
	}
	if q.Type != "" {
		db = db.Where("type = ?", q.Type)
	}
	if q.From != nil {
		db = db.Where("started_at >= ?", q.From)
	}
	if q.To != nil {
		db = db.Where("started_at < ?", q.To)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []*model.CareRecord
	err := db.Preload("Recorder").
		Order("started_at DESC").
		Offset((page - 1) * size).Limit(size).
		Find(&list).Error
	return list, total, err
}
