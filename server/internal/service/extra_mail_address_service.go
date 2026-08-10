package service

import (
	"errors"

	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

var (
	ErrMailAddressNotFound = errors.New("mail address not found")
	ErrMailAddressExists   = errors.New("mail address already exists")
)

type ExtraMailAddressService struct {
	db *gorm.DB
}

func NewExtraMailAddressService(db *gorm.DB) *ExtraMailAddressService {
	return &ExtraMailAddressService{db: db}
}

// ListExtraMailAddresses lists all extra mail addresses for a user
func (s *ExtraMailAddressService) ListExtraMailAddresses(userName string) ([]string, error) {
	var addresses []model.AccountExtraMailAddress
	if err := s.db.Where("user_name = ?", userName).Order("mail_address").Find(&addresses).Error; err != nil {
		return nil, err
	}

	result := make([]string, len(addresses))
	for i, addr := range addresses {
		result[i] = addr.MailAddress
	}

	return result, nil
}

// AddExtraMailAddress adds a new extra mail address
func (s *ExtraMailAddressService) AddExtraMailAddress(userName, mailAddress string) error {
	// Check if address already exists
	var count int64
	if err := s.db.Model(&model.AccountExtraMailAddress{}).
		Where("user_name = ? AND mail_address = ?", userName, mailAddress).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrMailAddressExists
	}

	addr := model.AccountExtraMailAddress{
		UserName:    userName,
		MailAddress: mailAddress,
	}
	return s.db.Create(&addr).Error
}

// DeleteExtraMailAddress deletes an extra mail address
func (s *ExtraMailAddressService) DeleteExtraMailAddress(userName, mailAddress string) error {
	result := s.db.Where("user_name = ? AND mail_address = ?", userName, mailAddress).
		Delete(&model.AccountExtraMailAddress{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrMailAddressNotFound
	}
	return nil
}

// GetPrimaryMailAddress gets the primary mail address for a user
func (s *ExtraMailAddressService) GetPrimaryMailAddress(userName string) (string, error) {
	var account model.Account
	if err := s.db.Where("user_name = ?", userName).First(&account).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", ErrMailAddressNotFound
		}
		return "", err
	}
	return account.MailAddress, nil
}
