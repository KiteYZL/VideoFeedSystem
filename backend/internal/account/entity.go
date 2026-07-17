package account

type Account struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Username     string `gorm:"type:varchar(64);uniqueIndex;not null" json:"username"`
	Passwd       string `json:"-"`
	Token        string `json:"-"`
	RefreshToken string `json:"-"`
	AvatarURL    string `gorm:"type:varchar(512)" json:"avatar_url,omitempty"`
	Bio          string `gorm:"type:varchar(255)" json:"bio,omitempty"`
}

type CreateAccountRequest struct {
	Username string `json:"username"`
	Passwd   string `json:"passwd"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Passwd   string `json:"passwd"`
}

type LoginResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	AccountID    uint   `json:"account_id"`
	Username     string `json:"username"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type RenameRequest struct {
	NewUsername string `json:"new_username"`
}

type FindByIDRequest struct {
	ID uint `json:"id"`
}

type FindByUsernameRequest struct {
	Username string `json:"username"`
}

type ChangePasswdRequest struct {
	Username  string `json:"username"`
	OldPasswd string `json:"old_passwd"`
	NewPasswd string `json:"new_passwd"`
}
