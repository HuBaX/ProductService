// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.18.0
// source: query.sql

package datahandling

import (
	"context"
	"database/sql"
)

const addProduct = `-- name: AddProduct :execresult
INSERT INTO product(details, name, price, category_id)
 VALUES(?, ?, ?, ?)
`

type AddProductParams struct {
	Details    string
	Name       string
	Price      float64
	CategoryID int32
}

func (q *Queries) AddProduct(ctx context.Context, arg AddProductParams) (sql.Result, error) {
	return q.db.ExecContext(ctx, addProduct,
		arg.Details,
		arg.Name,
		arg.Price,
		arg.CategoryID,
	)
}

const delProduct = `-- name: DelProduct :exec
DELETE FROM product WHERE id=?
`

func (q *Queries) DelProduct(ctx context.Context, id int32) error {
	_, err := q.db.ExecContext(ctx, delProduct, id)
	return err
}

const getProduct = `-- name: GetProduct :one
SELECT id, details, name, price, category_id FROM product WHERE id=?
`

func (q *Queries) GetProduct(ctx context.Context, id int32) (Product, error) {
	row := q.db.QueryRowContext(ctx, getProduct, id)
	var i Product
	err := row.Scan(
		&i.ID,
		&i.Details,
		&i.Name,
		&i.Price,
		&i.CategoryID,
	)
	return i, err
}

const getProductByName = `-- name: GetProductByName :many
SELECT id, details, name, price, category_id FROM product WHERE name=?
`

func (q *Queries) GetProductByName(ctx context.Context, name string) ([]Product, error) {
	rows, err := q.db.QueryContext(ctx, getProductByName, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Product
	for rows.Next() {
		var i Product
		if err := rows.Scan(
			&i.ID,
			&i.Details,
			&i.Name,
			&i.Price,
			&i.CategoryID,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getProducts = `-- name: GetProducts :many
SELECT id, details, name, price, category_id FROM product
`

func (q *Queries) GetProducts(ctx context.Context) ([]Product, error) {
	rows, err := q.db.QueryContext(ctx, getProducts)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Product
	for rows.Next() {
		var i Product
		if err := rows.Scan(
			&i.ID,
			&i.Details,
			&i.Name,
			&i.Price,
			&i.CategoryID,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}