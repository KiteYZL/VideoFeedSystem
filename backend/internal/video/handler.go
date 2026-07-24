package video

import (
	"net/http"

	"demo/internal/errorstatus"
	appjwt "demo/internal/middleware/jwt"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(errorstatus.ErrorToStatus(err), gin.H{"error": err.Error()})
		return
	}
	id, err := appjwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	name, err := appjwt.GetAccountUsername(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	item, err := h.service.CreateDraft(c, id, name, req)
	if err != nil {
		c.JSON(errorstatus.ErrorToStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) Publish(c *gin.Context)   { h.changeStatus(c, StatusPublished, true) }
func (h *Handler) Hide(c *gin.Context)      { h.changeStatus(c, StatusHidden, true) }
func (h *Handler) Delete(c *gin.Context)    { h.changeStatus(c, StatusDeleted, true) }
func (h *Handler) Republish(c *gin.Context) { h.changeStatus(c, StatusPublished, true) }

func (h *Handler) changeStatus(c *gin.Context, status Status, usePublish bool) {
	var req VideoIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(errorstatus.ErrorToStatus(err), gin.H{"error": err.Error()})
		return
	}
	id, err := appjwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	var item *Video
	if usePublish && status == StatusPublished {
		item, err = h.service.Publish(c, id, req.VideoID)
	} else {
		item, err = h.service.ChangeStatus(c, id, req.VideoID, status)
	}
	if err != nil {
		c.JSON(errorstatus.ErrorToStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) Update(c *gin.Context) {
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(errorstatus.ErrorToStatus(err), gin.H{"error": err.Error()})
		return
	}
	id, err := appjwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	item, err := h.service.Update(c, id, req)
	if err != nil {
		c.JSON(errorstatus.ErrorToStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) Detail(c *gin.Context) {
	var req VideoIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(errorstatus.ErrorToStatus(err), gin.H{"error": err.Error()})
		return
	}
	item, err := h.service.Get(c, req.VideoID)
	if err != nil {
		c.JSON(errorstatus.ErrorToStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) Feed(c *gin.Context) {
	var req FeedRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.service.Feed(c, req)
	if err != nil {
		c.JSON(errorstatus.ErrorToStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) Like(c *gin.Context)   { h.like(c, true) }
func (h *Handler) Unlike(c *gin.Context) { h.like(c, false) }

func (h *Handler) like(c *gin.Context, liked bool) {
	var req VideoIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(errorstatus.ErrorToStatus(err), gin.H{"error": err.Error()})
		return
	}
	userID, err := appjwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.Like(c, userID, req.VideoID, liked); err != nil {
		c.JSON(errorstatus.ErrorToStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"video_id": req.VideoID, "liked": liked})
}

func (h *Handler) LikeStatus(c *gin.Context) {
	var req VideoIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(errorstatus.ErrorToStatus(err), gin.H{"error": err.Error()})
		return
	}
	userID, err := appjwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	liked, err := h.service.GetLikeStatus(c, userID, req.VideoID)
	if err != nil {
		c.JSON(errorstatus.ErrorToStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, LikeStatusResponse{VideoID: req.VideoID, Liked: liked})
}

func (h *Handler) View(c *gin.Context) {
	var req VideoIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(errorstatus.ErrorToStatus(err), gin.H{"error": err.Error()})
		return
	}
	userID, err := appjwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.RecordView(c, userID, req.VideoID); err != nil {
		c.JSON(errorstatus.ErrorToStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"video_id": req.VideoID, "recorded": true})
}
