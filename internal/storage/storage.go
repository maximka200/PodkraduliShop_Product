package storage

import (
	"context"
	"fmt"
	"log/slog"
	"productservice/internal/config"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	productv1 "github.com/maximka200/protobuff_product/gen"
)

const (
	productTable = "products"
)

type StorageStruct struct {
	db  *sqlx.DB
	log *slog.Logger
}

func NewDB(cfg *config.Config) (*sqlx.DB, error) {
	op := "storage.NewSqlxDB"

	db, err := sqlx.Open("postgres",
		fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			cfg.DB.Host, cfg.DB.Port, cfg.DB.Username, cfg.DB.Password, cfg.DB.DBname, cfg.DB.SSLmode))
	if err != nil {
		return nil, fmt.Errorf("%s:%s", err, op)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("%s:%s", err, op)
	}

	return db, nil
}

func NewStorageStruct(db *sqlx.DB, log *slog.Logger) *StorageStruct {
	return &StorageStruct{db: db, log: log}
}

func (s *StorageStruct) NewProduct(ctx context.Context, imageURL string, title string, description string, discount uint8, price int64, currency int32, productURL string) (int64, error) {
	const op = "storage.NewProduct"

	s.log.Info(imageURL)
	stmt, err := s.db.Prepare(fmt.Sprintf("INSERT INTO %s (image_url, title, description, price, currency, discount, product_url) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id", productTable))
	if err != nil {
		s.log.Error(fmt.Sprintf("%s: %s", op, err))
		return 0, fmt.Errorf("%s: %s", op, err)
	}

	res, err := stmt.QueryContext(ctx, imageURL, title, description, price, currency, discount, productURL)
	if err != nil {
		s.log.Error(fmt.Sprintf("%s: %s", op, err))
		return 0, fmt.Errorf("%s: %s", op, err)
	}

	// INSERT 0 1
	res.Next()

	var id int64

	if err := res.Scan(&id); err != nil {
		return 0, fmt.Errorf("%s: %s", op, err)
	}

	return id, nil
}

func (s *StorageStruct) DeleteProduct(ctx context.Context, id int64) (bool, error) {
	const op = "storage.DeleteProduct"

	stmt, err := s.db.Prepare(fmt.Sprintf("DELETE FROM %s WHERE id=$1", productTable))
	if err != nil {
		s.log.Error(fmt.Sprintf("%s: %s", op, err))
		return false, fmt.Errorf("%s: %s", op, err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(id)
	if err != nil {
		return false, fmt.Errorf("%s: %s", op, err)
	}
	count, err := res.RowsAffected()
	if err != nil {
		s.log.Error(fmt.Sprintf("%s: %s", op, err))
		return false, fmt.Errorf("%s: %s", op, err)
	} else if count == 0 {
		s.log.Error(fmt.Sprintf("%s: do not delete element with id=%d", op, id))
		return false, nil
	}

	return true, nil
}

func (s *StorageStruct) GetProduct(ctx context.Context, id int64) (*productv1.GetProductResponse, error) {
	const op = "storage.GetProduct"

	model := &productv1.GetProductResponse{}

	stmt, err := s.db.Prepare(fmt.Sprintf("SELECT * FROM %s WHERE id=$1", productTable))
	if err != nil {
		s.log.Error(fmt.Sprintf("%s: %s", op, err))
		return nil, fmt.Errorf("%s: %s", op, err)
	}

	res := stmt.QueryRowContext(ctx, id)
	// expected 8 destination arguments in Scan, not 1
	if err := res.Scan(&model.Id, &model.ImageURL, &model.Title, &model.Description, &model.Price, &model.Currency, &model.Discount, &model.ProductURL); err != nil {
		s.log.Error(fmt.Sprintf("%s: %s", op, err))
		return nil, fmt.Errorf("%s: %s", op, err)
	}

	return model, nil
}

func (s *StorageStruct) GetAvailableId(ctx context.Context) (*[]int, error) {
	const op = "storage.GetAAvailableId"

	var idList []int
	stmt, err := s.db.Prepare(fmt.Sprintf("SELECT id FROM %s", productTable))
	if err != nil {
		s.log.Error(fmt.Sprintf("%s: %s", op, err))
		return nil, err
	}

	res, err := stmt.QueryContext(ctx)
	if err != nil {
		s.log.Error(fmt.Sprintf("%s: %s", op, err))
		return nil, err
	}

	for res.Next() {
		var id int
		if err := res.Scan(&id); err != nil {
			s.log.Error(fmt.Sprintf("%s: %s", op, err))
			return nil, err
		}
		idList = append(idList, id)
	}

	return &idList, nil
}
