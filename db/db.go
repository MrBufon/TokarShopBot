package db

import (
	"fmt"
	"os"
	"strings"

	"github.com/MrBufon/TokarShopBot/collections"

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

func FillPermissionMap(UserRights *collections.MutexMap[int64, string]) error {
	var rows []collections.UserRight

	err := DB.Table("user_rights").Select("id, permission").Find(&rows).Error
	if err != nil {
		return err
	}

	for _, r := range rows {
		UserRights.Set(r.ID, r.Permission)
	}

	return nil
}

func InsertIntoGoods(good collections.Good) error {
	return DB.Create(&good).Error
}

func DeleteFromGoods(order int64) error {
	result := DB.Where("id = ?", order).Delete(&collections.Good{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("товар с id %d не найден", order)
	}

	return nil
}

func FindInGoods(name string) (string, error) {
	var goods []collections.Good

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

func FindGoodStrInGoodsById(id int64) (string, error) {
	var good collections.Good

	err := DB.Where("id = ?", id).First(&good).Error

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Номер: %d | Наименование: %s | Количество: %d\n", good.Id, good.Name, good.Amount), nil
}

func FindGoodInGoodsById(id int64) (collections.Good, error) {
	var good collections.Good

	err := DB.Where("id = ?", id).First(&good).Error

	if err != nil {
		return collections.Good{}, err
	}

	return good, nil
}

func EditInGoods(updated collections.Good) error {
	result := DB.Model(&collections.Good{}).
		Where("id = ?", updated.Id).
		Select("amount", "name").
		Updates(updated)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("товар не найден")
	}

	return nil
}
