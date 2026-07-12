package account

import (
	"context"
	"demo/internal/auth"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

type AccountService struct {
	repo *AccountRepository
}

func NewAccountService(repo *AccountRepository) *AccountService {
	return &AccountService{repo: repo}
}

// bcrypt哈希加密密码并写入数据库
func (this *AccountService) CreateAccount(ctx context.Context, account *Account) error {
	if account.Passwd == "" {
		return errors.New("password can't be empty!")
	}

	hashedPasswd, err := bcrypt.GenerateFromPassword([]byte(account.Passwd), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	account.Passwd = string(hashedPasswd)

	if err := this.repo.CreateAccount(ctx, account); err != nil {
		return err
	}
	return nil
}

func (this *AccountService) Login(ctx context.Context, username string, password string) (string, string, error) {
	account, err := this.repo.FindByUsername(ctx, username)
	if err != nil {
		return "", "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(account.Passwd), []byte(password)); err != nil {
		return "", "", err
	}

	token, err := auth.GenerateToken(account.ID, username)
	if err != nil {
		return "", "", err
	}
	refreshToken, err := auth.GenerateRefreshToken()
	if err != nil {
		return "", "", err
	}

	if err := this.repo.Login(ctx, account.ID, token, refreshToken); err != nil {
		return "", "", err
	}

	return token, refreshToken, nil
}

func (this *AccountService) RefreshAccessToken(ctx context.Context, refreshToken string) (string, uint, string, error) {
	account, err := this.repo.FindByRefreshToken(ctx, refreshToken)
	if err != nil {
		return "", 0, "", err
	}

	newToken, err := auth.GenerateToken(account.ID, account.Username)
	if err != nil {
		return "", 0, "", err
	}
	if err := this.repo.UpdateToken(ctx, account.ID, newToken); err != nil {
		return "", 0, "", err
	}
	return newToken, account.ID, account.Username, nil
}

func (this *AccountService) Logout(ctx context.Context, id uint) error {
	return this.repo.Logout(ctx, id)
}

func (this *AccountService) FindByID(ctx context.Context, id uint) (*Account, error) {
	return this.repo.FindByID(ctx, id)
}

func (this *AccountService) FindByUsername(ctx context.Context, username string) (*Account, error) {
	return this.repo.FindByUsername(ctx, username)
}
