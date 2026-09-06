package controller

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/yantx/baby-care-workbench/backend/internal/middleware"
	"github.com/yantx/baby-care-workbench/backend/internal/model"
	"github.com/yantx/baby-care-workbench/backend/internal/repository"
	"github.com/yantx/baby-care-workbench/backend/internal/service"
)

// Controller HTTP 控制器
type Controller struct {
	Svc *service.Service
}

func New(svc *service.Service) *Controller {
	return &Controller{Svc: svc}
}

func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, model.APIResponse{Code: model.CodeOK, Message: "ok", Data: data})
}

func fail(c *gin.Context, bizErr *model.BusinessError) {
	c.JSON(http.StatusOK, model.APIResponse{Code: bizErr.Code, Message: bizErr.Message})
}

func serverError(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, model.APIResponse{
		Code: model.CodeServerError, Message: "服务器内部错误: " + err.Error(),
	})
}

// respondError 统一错误映射
func (ctl *Controller) respondError(c *gin.Context, err error) {
	var bizErr *model.BusinessError
	if errors.As(err, &bizErr) {
		fail(c, bizErr)
		return
	}
	switch {
	case errors.Is(err, model.ErrNotFound):
		c.JSON(http.StatusOK, model.APIResponse{Code: model.CodeNotFound, Message: "资源不存在"})
	case errors.Is(err, model.ErrForbidden):
		c.JSON(http.StatusForbidden, model.APIResponse{Code: model.CodeForbidden, Message: "无权访问该家庭数据"})
	default:
		serverError(c, err)
	}
}

func currentUserID(c *gin.Context) int64 {
	v, _ := c.Get(middleware.CtxUserID)
	id, _ := v.(int64)
	return id
}

// resolveFamilyID 取活跃家庭：优先 token claims，为 0 时回源用户表（建家庭/加入家庭后旧 Token 兜底）
func (ctl *Controller) resolveFamilyID(c *gin.Context) int64 {
	if v, ok := c.Get("family_id"); ok {
		if id, _ := v.(int64); id != 0 {
			return id
		}
	}
	user, err := ctl.Svc.Repo.GetUser(c.Request.Context(), currentUserID(c))
	if err != nil || user == nil {
		return 0
	}
	// 写回上下文，同一请求内复用
	c.Set("family_id", user.ActiveFamilyID)
	return user.ActiveFamilyID
}

// ---------- Auth ----------

// Login POST /api/v1/auth/login
func (ctl *Controller) Login(c *gin.Context) {
	var req model.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, model.NewBizError(model.CodeParamError, "参数错误: "+err.Error()))
		return
	}
	resp, err := ctl.Svc.Login(c.Request.Context(), req.Code, req.Nickname)
	if err != nil {
		ctl.respondError(c, err)
		return
	}
	ok(c, resp)
}

// ---------- User ----------

// Me GET /api/v1/user/me
func (ctl *Controller) Me(c *gin.Context) {
	user, err := ctl.Svc.Repo.GetUser(c.Request.Context(), currentUserID(c))
	if err != nil {
		ctl.respondError(c, err)
		return
	}
	ok(c, user)
}

// UpdateProfile PUT /api/v1/user/me
func (ctl *Controller) UpdateProfile(c *gin.Context) {
	var req model.UpdateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, model.NewBizError(model.CodeParamError, "参数错误: "+err.Error()))
		return
	}
	user, err := ctl.Svc.UpdateProfile(c.Request.Context(), currentUserID(c), &req)
	if err != nil {
		ctl.respondError(c, err)
		return
	}
	ok(c, user)
}

// ---------- Family ----------

// CreateFamily POST /api/v1/families
func (ctl *Controller) CreateFamily(c *gin.Context) {
	var req model.CreateFamilyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, model.NewBizError(model.CodeParamError, "参数错误: "+err.Error()))
		return
	}
	resp, err := ctl.Svc.CreateFamily(c.Request.Context(), currentUserID(c), &req)
	if err != nil {
		ctl.respondError(c, err)
		return
	}
	ok(c, resp)
}

// JoinFamily POST /api/v1/families/join
func (ctl *Controller) JoinFamily(c *gin.Context) {
	var req model.JoinFamilyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, model.NewBizError(model.CodeParamError, "参数错误: "+err.Error()))
		return
	}
	resp, err := ctl.Svc.JoinFamily(c.Request.Context(), currentUserID(c), &req)
	if err != nil {
		ctl.respondError(c, err)
		return
	}
	ok(c, resp)
}

// CurrentFamily GET /api/v1/families/current
func (ctl *Controller) CurrentFamily(c *gin.Context) {
	familyID := ctl.resolveFamilyID(c)
	if familyID == 0 {
		fail(c, model.NewBizError(model.CodeNotFound, "尚未创建或加入家庭"))
		return
	}
	resp, err := ctl.Svc.GetFamilyDetail(c.Request.Context(), currentUserID(c), familyID)
	if err != nil {
		ctl.respondError(c, err)
		return
	}
	ok(c, resp)
}

// ---------- Baby ----------

// CreateBaby POST /api/v1/babies
func (ctl *Controller) CreateBaby(c *gin.Context) {
	var req model.CreateBabyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, model.NewBizError(model.CodeParamError, "参数错误: "+err.Error()))
		return
	}
	baby, err := ctl.Svc.CreateBaby(c.Request.Context(), currentUserID(c), ctl.resolveFamilyID(c), &req)
	if err != nil {
		ctl.respondError(c, err)
		return
	}
	ok(c, baby)
}

// ---------- CareRecord ----------

// CreateRecord POST /api/v1/records
func (ctl *Controller) CreateRecord(c *gin.Context) {
	var req model.CareRecordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, model.NewBizError(model.CodeParamError, "参数错误: "+err.Error()))
		return
	}
	rec, err := ctl.Svc.CreateRecord(c.Request.Context(), currentUserID(c), ctl.resolveFamilyID(c), &req)
	if err != nil {
		ctl.respondError(c, err)
		return
	}
	ok(c, rec)
}

// UpdateRecord PUT /api/v1/records/:id
func (ctl *Controller) UpdateRecord(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req model.CareRecordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, model.NewBizError(model.CodeParamError, "参数错误: "+err.Error()))
		return
	}
	rec, err := ctl.Svc.UpdateRecord(c.Request.Context(), currentUserID(c), ctl.resolveFamilyID(c), id, &req)
	if err != nil {
		ctl.respondError(c, err)
		return
	}
	ok(c, rec)
}

// DeleteRecord DELETE /api/v1/records/:id
func (ctl *Controller) DeleteRecord(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := ctl.Svc.DeleteRecord(c.Request.Context(), currentUserID(c), ctl.resolveFamilyID(c), id); err != nil {
		ctl.respondError(c, err)
		return
	}
	ok(c, nil)
}

// ListRecords GET /api/v1/records?date=2026-09-06&type=feeding&page=1&page_size=20
func (ctl *Controller) ListRecords(c *gin.Context) {
	p := model.Pagination{}
	_ = c.ShouldBindQuery(&p)
	p.Normalize()

	q := parseRecordQuery(c)
	records, total, err := ctl.Svc.ListRecords(c.Request.Context(), currentUserID(c), ctl.resolveFamilyID(c), q, p.Page, p.PageSize)
	if err != nil {
		ctl.respondError(c, err)
		return
	}
	ok(c, gin.H{"list": records, "page": p.Page, "page_size": p.PageSize, "total": total})
}

// TodayStats GET /api/v1/stats/today
func (ctl *Controller) TodayStats(c *gin.Context) {
	babyID, _ := strconv.ParseInt(c.Query("baby_id"), 10, 64)
	resp, err := ctl.Svc.TodayStats(c.Request.Context(), currentUserID(c), ctl.resolveFamilyID(c), babyID)
	if err != nil {
		ctl.respondError(c, err)
		return
	}
	ok(c, resp)
}

// parseRecordQuery 解析查询参数：date（当日）或 from/to（范围）
func parseRecordQuery(c *gin.Context) (q repository.RecordQuery) {
	q.Type = c.Query("type")
	q.BabyID, _ = strconv.ParseInt(c.Query("baby_id"), 10, 64)

	if date := c.Query("date"); date != "" {
		if day, err := time.ParseInLocation("2006-01-02", date, time.Local); err == nil {
			q.From, q.To = day, day.Add(24*time.Hour)
		}
		return
	}
	if from := c.Query("from"); from != "" {
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", from, time.Local); err == nil {
			q.From = t
		}
	}
	if to := c.Query("to"); to != "" {
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", to, time.Local); err == nil {
			q.To = t
		}
	}
	return
}
