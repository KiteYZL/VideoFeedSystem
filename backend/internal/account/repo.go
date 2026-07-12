package account

import (
	"context"

	"gorm.io/gorm"
)

type AccountRepository struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (this *AccountRepository) CreateAccount(ctx context.Context, acc *Account) error {
	return this.db.WithContext(ctx).Create(acc).Error
}

func (this *AccountRepository) Login(ctx context.Context, id uint, token, refreshToken string) error {
	return this.db.WithContext(ctx).Model(&Account{}).Where("id = ?", id).Updates(map[string]interface{}{"token": token, "refresh_token": refreshToken}).Error
}

func (this *AccountRepository) Logout(ctx context.Context, id uint) error {
	return this.db.WithContext(ctx).Model(&Account{}).Where("id = ?", id).Updates(map[string]interface{}{"token": "", "refresh_token": ""}).Error
}

func (this *AccountRepository) UpdateToken(ctx context.Context, id uint, token string) error {
	return this.db.WithContext(ctx).Model(&Account{}).Where("id = ?", id).Update("token", token).Error
}

func (this *AccountRepository) UpdateRefreshToken(ctx context.Context, id uint, refreshToken string) error {
	return this.db.WithContext(ctx).Model(&Account{}).Where("id = ?", id).Update("refresh_token", refreshToken).Error
}

func (this *AccountRepository) FindByID(ctx context.Context, id uint) (*Account, error) {
	var account Account
	if err := this.db.WithContext(ctx).First(&account, id).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (this *AccountRepository) FindByUsername(ctx context.Context, username string) (*Account, error) {
	var account Account
	if err := this.db.WithContext(ctx).Where("username = ?", username).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (this *AccountRepository) FindByRefreshToken(ctx context.Context, refreshToken string) (*Account, error) {
	var account Account
	if err := this.db.WithContext(ctx).Where("refresh_token = ?", refreshToken).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}
