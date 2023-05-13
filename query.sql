-- name: AddProduct :execresult
INSERT INTO product(details, name, price, category_id)
 VALUES(?, ?, ?, ?);

-- name: GetProduct :one
SELECT * FROM product WHERE id=?;

-- name: GetProducts :many
SELECT * FROM product;

-- name: GetProductByName :many
SELECT * FROM product WHERE name=?;

-- name: DelProduct :exec
DELETE FROM product WHERE id=?;