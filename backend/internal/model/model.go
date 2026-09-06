package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// BaseModel 通用基础字段
type BaseModel struct {
	ID        int64     `gorm:"primaryKey;bigserial" json:"id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// User 用户（微信登录）
type User struct {
	BaseModel
	OpenID    string `gorm:"column:openid;uniqueIndex;size:64;not null" json:"-"`
	UnionID   string `gorm:"column:unionid;size:64" json:"-"`
	Nickname  string `gorm:"size:64" json:"nickname"`
	AvatarURL string `gorm:"size:512" json:"avatar_url"`
	// 用户当前活跃的家庭（冗余字段，登录后选择）
	ActiveFamilyID int64 `gorm:"index" json:"-"`
}

func (User) TableName() string { return "users" }

// Family 家庭
type Family struct {
	BaseModel
	Name       string `gorm:"size:64;not null" json:"name"`
	InviteCode string `gorm:"column:invite_code;uniqueIndex;size:8;not null" json:"invite_code"`
	OwnerID    int64  `gorm:"index;not null" json:"owner_id"`
}

func (Family) TableName() string { return "families" }

// 家庭成员角色
const (
	RoleFather       = "father"
	RoleMother       = "mother"
	RoleGrandparent  = "grandparent"
	RoleOther        = "other"
)

// FamilyMember 家庭成员（用户与家庭多对多，含角色）
type FamilyMember struct {
	BaseModel
	FamilyID int64  `gorm:"uniqueIndex:idx_family_user;not null" json:"family_id"`
	UserID   int64  `gorm:"uniqueIndex:idx_family_user;not null" json:"user_id"`
	Role     string `gorm:"size:16;not null" json:"role"` // father/mother/grandparent/other
	Nickname string `gorm:"size:64" json:"nickname"`       // 在家庭内的称呼，如"宝宝妈妈"
	JoinedAt time.Time `gorm:"autoCreateTime" json:"joined_at"`

	// 关联（查询时预加载）
	User   *User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Family *Family `gorm:"foreignKey:FamilyID" json:"family,omitempty"`
}

func (FamilyMember) TableName() string { return "family_members" }

// Baby 宝宝档案
type Baby struct {
	BaseModel
	FamilyID  int64  `gorm:"index;not null" json:"family_id"`
	Name      string `gorm:"size:64;not null" json:"name"`
	Gender    int    `gorm:"default:0" json:"gender"` // 0未知 1男 2女
	Birthday  *time.Time `json:"birthday"`
	AvatarURL string `gorm:"size:512" json:"avatar_url"`
}

func (Baby) TableName() string { return "babies" }

// 护理记录类型
const (
	RecordTypeFeeding     = "feeding"     // 喂奶
	RecordTypeSleep       = "sleep"       // 睡眠
	RecordTypeDiaper      = "diaper"      // 换尿布
	RecordTypeTemperature = "temperature" // 体温
	RecordTypeMedicine    = "medicine"    // 用药
)

// CareRecord 护理记录（核心表：喂奶/睡眠/尿布/体温/用药 统一记录）
type CareRecord struct {
	BaseModel
	FamilyID   int64      `gorm:"index:idx_family_started;not null" json:"family_id"`
	BabyID     int64      `gorm:"index;not null" json:"baby_id"`
	RecorderID int64      `gorm:"not null" json:"recorder_id"`
	Type       string     `gorm:"size:16;not null" json:"type"`
	StartedAt  time.Time  `gorm:"index:idx_family_started;not null" json:"started_at"`
	EndedAt    *time.Time `json:"ended_at,omitempty"` // 睡眠等有时长的记录
	AmountML   *float64   `gorm:"type:numeric(8,1)" json:"amount_ml,omitempty"` // 奶量
	Content    string     `gorm:"size:128" json:"content"`    // 类型化内容：尿布(湿/脏)、喂奶(亲喂/瓶喂/左/右)、药品名
	TempValue  *float64   `gorm:"type:numeric(4,1)" json:"temp_value,omitempty"` // 体温℃
	Note       string     `gorm:"size:512" json:"note"`
	Details    MapJSON    `gorm:"type:jsonb" json:"details,omitempty"` // 扩展明细

	Recorder *User `gorm:"foreignKey:RecorderID" json:"recorder,omitempty"`
}

func (CareRecord) TableName() string { return "care_records" }

// MapJSON JSONB 字段类型
type MapJSON map[string]any

func (m MapJSON) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	b, err := json.Marshal(m)
	return string(b), err
}

func (m *MapJSON) Scan(value any) error {
	if value == nil {
		*m = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return fmt.Errorf("MapJSON.Scan: 不支持的类型 %T", value)
	}
	return json.Unmarshal(bytes, m)
}
