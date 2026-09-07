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

type addMemberRequest struct {
	MemberID int64 `json:"member_id" binding:"required"`
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

// AddMember godoc
// @Summary      添加群成员
// @Description  向群组添加一名新成员，仅群主或管理员可操作
// @Tags         群组
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        groupID  path  int64  true  "群组ID"
// @Param        body     body  addMemberRequest  true  "成员信息"
// @Success      200  {object}  ApiResponse  "添加成功"
// @Failure      400  {object}  ApiResponse  "参数错误"
// @Failure      403  {object}  ApiResponse  "无权限"
// @Failure      404  {object}  ApiResponse  "群组不存在"
// @Failure      409  {object}  ApiResponse  "已是成员或群组已满"
// @Router       /group/{groupID}/member [post]
// AddMember handles POST /group/:groupID/member.
func (h *GroupHandler) AddMember(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("groupID"), 10, 64)
	if err != nil {
		Error(c, http.StatusBadRequest, CodeInvalidParam, "invalid group_id")
		return
	}

	var req addMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, CodeMissingParam, "member_id is required")
		return
	}

	userID := c.GetInt64("userID")
	if userID == 0 {
		Error(c, http.StatusUnauthorized, CodeUnauthorized, "unauthorized")
		return
	}

	err = h.groupSvc.AddMember(c.Request.Context(), groupID, userID, req.MemberID)
	if err != nil {
		switch err.Error() {
		case service.ErrNotOwnerOrAdmin:
			ServiceError(c, http.StatusForbidden, err.Error())
		case service.ErrGroupNotFound:
			ServiceError(c, http.StatusNotFound, err.Error())
		case service.ErrAlreadyMember:
			ServiceError(c, http.StatusConflict, err.Error())
		case service.ErrGroupFull:
			ServiceError(c, http.StatusConflict, err.Error())
		case service.ErrUserNotFound:
			ServiceError(c, http.StatusNotFound, err.Error())
		case service.ErrMemberNotFound:
			ServiceError(c, http.StatusNotFound, err.Error())
		case service.ErrMemberNotFriend:
			ServiceError(c, http.StatusForbidden, err.Error())
		default:
			Error(c, http.StatusInternalServerError, CodeInternalError, "internal error")
		}
		return
	}
	// ToDo
	// if h.cm != nil {
	// 	if client, ok := h.cm.Get(req.MemberID); ok {
	// 		group, getErr := h.groupSvc.GetGroupInfo(c.Request.Context(), groupID)
	// 		if getErr == nil && group != nil {
	// 			notice, _ := json.Marshal(map[string]interface{}{"type": "groupAdded", "data": map[string]interface{}{"groupId": groupID, "name": group.Name}})
	// 			select {
	// 			case client.SendCh <- notice:
	// 			default:
	// 			}
	// 		}
	// 	}
	// }

	SuccessMessage(c, "member added")
}

// RemoveMember godoc
// @Summary      移除群成员
// @Description  从群组中移除一名成员；若移除自己则为退群，移除他人则为踢出（仅群主/管理员可踢人）
// @Tags         群组
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        groupID   path  int64  true  "群组ID"
// @Param        memberID  path  int64  true  "成员ID"
// @Success      200  {object}  ApiResponse  "移除成功"
// @Failure      400  {object}  ApiResponse  "参数错误"
// @Failure      403  {object}  ApiResponse  "无权限"
// @Failure      404  {object}  ApiResponse  "群组不存在"
// @Router       /group/{groupID}/member/{memberID} [delete]
// RemoveMember handles DELETE /group/:groupID/member/:memberID.
// If the requesting user is the member being removed, it's a self-leave.
// Otherwise, it's a kick (owner/admin removes another member).
func (h *GroupHandler) RemoveMember(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("groupID"), 10, 64)
	if err != nil {
		Error(c, http.StatusBadRequest, CodeInvalidParam, "invalid group_id")
		return
	}

	memberID, err := strconv.ParseInt(c.Param("memberID"), 10, 64)
	if err != nil {
		Error(c, http.StatusBadRequest, CodeInvalidParam, "invalid member_id")
		return
	}

	userID := c.GetInt64("userID")
	if userID == 0 {
		Error(c, http.StatusUnauthorized, CodeUnauthorized, "unauthorized")
		return
	}

	err = h.groupSvc.RemoveMember(c.Request.Context(), groupID, userID, memberID)
	if err != nil {
		switch err.Error() {
		case service.ErrNotOwnerOrAdmin:
			ServiceError(c, http.StatusForbidden, err.Error())
		case service.ErrGroupNotFound:
			ServiceError(c, http.StatusNotFound, err.Error())
		case service.ErrCannotRemoveOwner, service.ErrCannotRemovePeer:
			ServiceError(c, http.StatusForbidden, err.Error())
		case service.ErrMemberNotFound:
			ServiceError(c, http.StatusNotFound, err.Error())
		default:
			Error(c, http.StatusInternalServerError, CodeInternalError, "internal error")
		}
		return
	}
	// if memberID != userID && h.cm != nil {
	// 	if client, ok := h.cm.Get(memberID); ok {
	// 		notice, _ := json.Marshal(map[string]interface{}{"type": "groupRemoved", "data": map[string]interface{}{"groupId": groupID, "reason": "removed"}})
	// 		select {
	// 		case client.SendCh <- notice:
	// 		default:
	// 		}
	// 	}
	// }

	SuccessMessage(c, "member removed")
}

// LeaveGroup godoc
// @Summary      退出群组
// @Description  当前用户退出指定群组，群主不能直接退群（需先转让群主）
// @Tags         群组
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        groupID  path  int64  true  "群组ID"
// @Success      200  {object}  ApiResponse  "退出成功"
// @Failure      400  {object}  ApiResponse  "参数错误"
// @Failure      403  {object}  ApiResponse  "群主不能退群"
// @Failure      404  {object}  ApiResponse  "群组不存在"
// @Router       /group/{groupID}/leave [post]
// LeaveGroup handles POST /group/:groupID/leave.
func (h *GroupHandler) LeaveGroup(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("groupID"), 10, 64)
	if err != nil {
		Error(c, http.StatusBadRequest, CodeInvalidParam, "invalid group_id")
		return
	}

	userID := c.GetInt64("userID")
	if userID == 0 {
		Error(c, http.StatusUnauthorized, CodeUnauthorized, "unauthorized")
		return
	}

	err = h.groupSvc.LeaveGroup(c.Request.Context(), groupID, userID)
	if err != nil {
		switch err.Error() {
		case service.ErrCannotLeaveAsOwner:
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

	SuccessMessage(c, "left group")
}

// RegisterRoutes registers all group HTTP routes on the given Gin router group.
func (h *GroupHandler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/group")
	g.POST("", h.CreateGroup)
	g.GET("/list", h.ListMyGroups)
	g.PUT("/:groupID", h.UpdateGroup)
	g.GET("/:groupID", h.GetGroupInfo)
	g.POST("/:groupID/member", h.AddMember)
	g.DELETE("/:groupID/member/:memberID", h.RemoveMember)
	g.POST("/:groupID/leave", h.LeaveGroup)
}
