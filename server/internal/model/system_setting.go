package model

// SystemSetting represents a system setting
type SystemSetting struct {
	Key   string `gorm:"primaryKey;column:key" json:"key"`
	Value string `gorm:"column:value" json:"value"`
}

func (SystemSetting) TableName() string { return "system_settings" }
