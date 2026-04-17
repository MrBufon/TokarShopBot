package db

import (
	"fmt"
	"os"
	"strings"

	"github.com/MrBufon/TokarShopBot/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init() error {
	dsn := os.Getenv("DB_DSN")

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	return err
}

func FillPermissionMap(UserRights map[int64]string) error {
	var rows []models.UserRight

	err := DB.Table("user_rights").Select("id, permission").Find(&rows).Error
	if err != nil {
		return err
	}

	for _, r := range rows {
		UserRights[r.ID] = r.Permission
	}

	return nil
}

func InsertIntoGoods(good models.Good) error {
	return DB.Create(&good).Error
}

func DeleteFromGoods(order int64) error {
	return DB.Where("id = ?", order).Delete(&models.Good{}).Error
}

func FindInGoods(name string) (string, error) {
	var goods []models.Good

	switch name {
	case "Всё":
		err := DB.Find(&goods).Error
		if err != nil {
			return "", err
		} else if len(goods) == 0 {
			return "", nil
		}
	default:
		err := DB.Where("name ILIKE ?", "%"+name+"%").Find(&goods).Error
		if err != nil {
			return "", err
		} else if len(goods) == 0 {
			return "", nil
		}
	}

	var sb strings.Builder
	for _, good := range goods {
		sb.WriteString(fmt.Sprintf("Номер: %d | Наименование: %s | Количество: %d\n", good.Id, good.Name, good.Amount))
	}
	return sb.String(), nil
}
