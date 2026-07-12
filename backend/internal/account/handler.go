package account

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AccountHandler struct {
	accountService *AccountService
}

func NewAccountHandler(accountService *AccountService) *AccountHandler {
	return &AccountHandler{accountService: accountService}
}

func (this *AccountHandler) CreateAccount(c *gin.Context) {
	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := this.accountService.CreateAccount(c.Request.Context(), &Account{
		Username: req.Username,
		Passwd:   req.Passwd,
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "注册失败，请稍后重试"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "account is successfully created"})
}

func (this *AccountHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	account, err := this.accountService.FindByUsername(c.Request.Context(), req.Username)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, refreshToken, err := this.accountService.Login(c.Request.Context(), req.Username, req.Passwd)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		AccountID:    account.ID,
		Username:     account.Username,
	})
}

func (this *AccountHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, accountID, username, err := this.accountService.RefreshAccessToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token:     token,
		AccountID: accountID,
		Username:  username,
	})
}

func (this *AccountHandler) Logout(c *gin.Context) {
	accountID, err := getAccountID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := this.accountService.Logout(c.Request.Context(), accountID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logout successfully"})
}

func (this *AccountHandler) FindByID(c *gin.Context) {
	var req FindByIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	account, err := this.accountService.FindByID(c.Request.Context(), req.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token:        account.Token,
		RefreshToken: account.RefreshToken,
		AccountID:    req.ID,
		Username:     account.Username,
	})
}

func (this *AccountHandler) FindByUsername(c *gin.Context) {
	var req FindByUsernameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	account, err := this.accountService.FindByUsername(c.Request.Context(), req.Username)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token:        account.Token,
		RefreshToken: account.RefreshToken,
		AccountID:    account.ID,
		Username:     req.Username,
	})
}

func getAccountID(c *gin.Context) (uint, error) {
	value, exists := c.Get("account_id")
	if !exists {
		return 0, errors.New("account_id not found in gin.Context")
	}

	accountID, ok := value.(uint)
	if !ok {
		return 0, errors.New("account_id has invalid type in gin.Context")
	}

	return accountID, nil
}
