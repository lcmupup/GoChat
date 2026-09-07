package api

import (
	"gochat/internal/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GroupHandler 提供群组端点的 Gin HTTP 处理程序。
type GroupHandler struct {
	groupSvc *service.GroupService
	// cm       *conn.ConnectionManager
}

// NewGroupHandler 创建一个包装了给定 GroupService 的 GroupHandler。
func NewGroupHandler(groupSvc *service.GroupService) *GroupHandler {
	// var cm *conn.ConnectionManager
	// if len(managers) > 0 {
	// 	cm = managers[0]
	// }
	return &GroupHandler{groupSvc: groupSvc}
}

// ── Request / response DTOs ──

type createGroupRequest struct {
	Name   string `json:"name" binding:"required"`
	Notice string `json:"notice"`
}

type createGroupResponse struct {
	GroupID int64 `json:"group_id"`
}

type updateGroupRequest struct {
	Name   string `json:"name" binding:"required"`
	Notice string `json:"notice"`
}

// ── Handlers ──

// CreateGroup godoc
// @Summary      创建群组
// @Description  创建一个新群组，创建者自动成为群主
// @Tags         群组
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  createGroupRequest  true  "群组信息"
// @Success      201   {object}  ApiResponse{data=createGroupResponse}  "创建成功"
// @Failure      400   {object}  ApiResponse  "参数错误"
// @Failure      401   {object}  ApiResponse  "未授权"
// @Router       /group [post]
// CreateGroup handles POST /group.
func (h *GroupHandler) CreateGroup(c *gin.Context) {
	var req createGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, CodeMissingParam, "name is required")
		return
	}

	userID := c.GetInt64("userID")
	if userID == 0 { // 该用户为通过JWT认证
		Error(c, http.StatusUnauthorized, CodeUnauthorized, "unauthorized")
		return
	}

	groupID, err := h.groupSvc.CreateGroup(c.Request.Context(), userID, req.Name, req.Notice)
	if err != nil {
		Error(c, http.StatusInternalServerError, CodeInternalError, "internal error")
		return
	}

	SuccessCreated(c, createGroupResponse{GroupID: groupID})
}

// UpdateGroup godoc
// @Summary      更新群组信息
// @Description  更新群组的名称和公告，仅群主或管理员可操作
// @Tags         群组
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        groupID  path  int64  true  "群组ID"
// @Param        body     body  updateGroupRequest  true  "更新群组信息"
// @Success      200  {object}  ApiResponse  "更新成功"
// @Failure      400  {object}  ApiResponse  "参数错误"
// @Failure      403  {object}  ApiResponse  "无权限"
// @Failure      404  {object}  ApiResponse  "群组不存在"
// @Router       /group/{groupID} [put]
// UpdateGroup handles PUT /group/:groupID.
func (h *GroupHandler) UpdateGroup(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("groupID"), 10, 64)
	if err != nil {
		Error(c, http.StatusBadRequest, CodeInvalidParam, "invalid group_id")
		return
	}

	var req updateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, CodeMissingParam, "name is required")
		return
	}

	userID := c.GetInt64("userID")
	if userID == 0 {
		Error(c, http.StatusUnauthorized, CodeUnauthorized, "unauthorized")
		return
	}

	err = h.groupSvc.UpdateGroup(c.Request.Context(), userID, groupID, req.Name, req.Notice)
	if err != nil {
		switch err.Error() {
		case service.ErrNotOwnerOrAdmin:
			ServiceError(c, http.StatusForbidden, err.Error())
		case service.ErrGroupNotFound:
			ServiceError(c, http.StatusNotFound, err.Error())
		case service.ErrMemberNotFound:
			ServiceError(c, http.StatusNotFound, err.Error())
		default:
			Error(c, http.StatusInternalServerError, CodeInternalError, "internal error")
		}
		return
	}

	SuccessMessage(c, "group updated")
}

// GetGroupInfo godoc
// @Summary      获取群组信息
// @Description  根据群组ID获取群组详细信息
// @Tags         群组
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        groupID  path  int64  true  "群组ID"
// @Success      200  {object}  ApiResponse{data=object}  "查询成功"
// @Failure      400  {object}  ApiResponse  "参数错误"
// @Failure      404  {object}  ApiResponse  "群组不存在"
// @Router       /group/{groupID} [get]
// GetGroupInfo handles GET /group/:groupID.
func (h *GroupHandler) GetGroupInfo(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("groupID"), 10, 64)
	if err != nil {
		Error(c, http.StatusBadRequest, CodeInvalidParam, "invalid group_id")
		return
	}

	group, err := h.groupSvc.GetGroupInfo(c.Request.Context(), groupID)
	if err != nil {
		switch err.Error() {
		case service.ErrGroupNotFound:
			ServiceError(c, http.StatusNotFound, err.Error())
		default:
			Error(c, http.StatusInternalServerError, CodeInternalError, "internal error")
		}
		return
	}

	Success(c, group)
}

func (h *GroupHandler) ListMyGroups(c *gin.Context) {
	groups, err := h.groupSvc.ListUserGroups(c.Request.Context(), c.GetInt64("userID"))
	if err != nil {
		Error(c, http.StatusInternalServerError, CodeInternalError, "internal error")
		return
	}
	Success(c, groups)
}

// RegisterRoutes registers all group HTTP routes on the given Gin router group.
func (h *GroupHandler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/group")
	g.POST("", h.CreateGroup)
	g.GET("/list", h.ListMyGroups)
	g.PUT("/:groupID", h.UpdateGroup)
	g.GET("/:groupID", h.GetGroupInfo)
}
