package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"vsmlab/productservice/datahandling"

	"encoding/json"
	"os"

	"github.com/go-sql-driver/mysql"
)

func main() {
	dbUser := os.Getenv("MYSQL_USER")
	dbAddr := os.Getenv("MYSQL_ADDRESS")
	dbPassword := os.Getenv("MYSQL_PASSWORD")
	dbName := os.Getenv("MYSQL_DATABASE")

	cfg := mysql.Config{
		User:                 dbUser,
		Passwd:               dbPassword,
		Net:                  "tcp",
		Addr:                 dbAddr,
		DBName:               dbName,
		AllowNativePasswords: true,
	}

	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	queries := datahandling.New(db)

	http.HandleFunc("/addProduct", handleAddProduct(ctx, queries))
	http.HandleFunc("/getProduct", handleGetProductById(ctx, queries))
	http.HandleFunc("/getProducts", handleGetProducts(ctx, queries))
	http.HandleFunc("/getProductByName", handleGetProductByName(ctx, queries))
	http.HandleFunc("/getProductsBySearchValues", handleGetProductsBySearchValues(ctx, db))
	http.HandleFunc("/delProductById", handleDelProductById(ctx, queries))

	http.ListenAndServe("0.0.0.0:8082", nil)
}

func handleAddProduct(ctx context.Context, queries *datahandling.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var nameJSON map[string]any
		err := readJSON(r, &nameJSON)
		defer r.Body.Close()
		if err != nil {
			setError(w, ErrReadJSON)
			return
		}

		name := nameJSON["name"].(string)
		price := nameJSON["price"].(float64)
		catId := nameJSON["categoryId"].(int32)
		details := nameJSON["details"].(string)

		product := datahandling.AddProductParams{
			Name:       name,
			Price:      price,
			CategoryID: catId,
			Details:    details,
		}

		resp, err := http.Get("category-service/getCategory?id=" + strconv.Itoa(int(catId)))

		if err != nil {
			setError(w, ErrRequestFailed)
			return
		}

		if resp.StatusCode == http.StatusOK {
			_, err = queries.AddProduct(ctx, product)
			if err != nil {
				setError(w, ErrQueryDatabase)
				return
			}
		}

		w.WriteHeader(http.StatusNotFound)
	}
}

func handleGetProductById(ctx context.Context, queries *datahandling.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var idJSON map[string]any
		err := readJSON(r, &idJSON)
		defer r.Body.Close()
		if err != nil {
			setError(w, ErrReadJSON)
			return
		}

		id := idJSON["id"].(int32)
		product, err := queries.GetProduct(ctx, id)
		if err != nil {
			setError(w, ErrQueryDatabase)
			return
		}

		err = writeJSON(w, http.StatusOK, product)
		if err != nil {
			setError(w, ErrWriteJSON)
		}
	}
}

func handleGetProducts(ctx context.Context, queries *datahandling.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		products, err := queries.GetProducts(ctx)
		if err != nil {
			fmt.Println(err.Error())
			setError(w, ErrQueryDatabase)
			return
		}

		productMap := map[string][]datahandling.Product{
			"products": products,
		}

		err = writeJSON(w, http.StatusOK, productMap)
		if err != nil {
			setError(w, ErrWriteJSON)
		}
	}
}

func handleGetProductByName(ctx context.Context, queries *datahandling.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var nameJSON map[string]any
		err := readJSON(r, &nameJSON)
		defer r.Body.Close()
		if err != nil {
			setError(w, ErrReadJSON)
			return
		}

		name := nameJSON["name"].(string)
		product, err := queries.GetProductByName(ctx, name)
		if err != nil {
			setError(w, ErrQueryDatabase)
			return
		}

		err = writeJSON(w, http.StatusOK, product)
		if err != nil {
			setError(w, ErrWriteJSON)
		}
	}
}

func handleGetProductsBySearchValues(ctx context.Context, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var nameJSON map[string]any
		err := readJSON(r, &nameJSON)
		defer r.Body.Close()
		if err != nil {
			setError(w, ErrReadJSON)
			return
		}

		details := nameJSON["searchDescription"].(string)
		minPrice := nameJSON["SearchMinPrice"].(float32)
		maxPrice := nameJSON["SearchMaxPrice"].(float32)
		if details == "" && minPrice == 0 && maxPrice == 0 {
			setError(w, ErrNoSearchParamSet)
			return
		}

		sql := "SELECT * FROM products WHERE 1=1"
		args := []any{}

		if minPrice > 0 {
			sql += "AND price >= ?"
			args = append(args, minPrice)
		}

		if maxPrice > 0 {
			sql += "AND price <= ?"
			args = append(args, maxPrice)
		}

		if details != "" {
			sql += "AND details LIKE '%'?'%'"
			args = append(args, details)
		}

		stmt, err := db.Prepare(sql)
		if err != nil {
			setError(w, ErrPrepareQuery)
			return
		}
		defer stmt.Close()

		rows, err := stmt.Query(args)
		if err != nil {
			setError(w, ErrQueryDatabase)
			return
		}
		defer rows.Close()

		var products []datahandling.Product

		for rows.Next() {
			var p datahandling.Product
			err = rows.Scan(&p.ID, &p.Details, &p.Name, &p.Price, &p.CategoryID)
			if err != nil {
				setError(w, ErrRowScan)
				return
			}
			products = append(products, p)
		}

		productMap := map[string][]datahandling.Product{
			"products": products,
		}

		err = writeJSON(w, http.StatusOK, productMap)
		if err != nil {
			setError(w, ErrWriteJSON)
		}
	}
}

func handleDelProductById(ctx context.Context, queries *datahandling.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var idJSON map[string]any
		err := readJSON(r, &idJSON)
		defer r.Body.Close()
		if err != nil {
			setError(w, ErrReadJSON)
			return
		}

		id := idJSON["id"].(int32)
		err = queries.DelProduct(ctx, id)
		if err != nil {
			setError(w, ErrQueryDatabase)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) error {
	w.WriteHeader(status)
	w.Header().Add("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(value)
}

func readJSON(r *http.Request, value *map[string]any) error {
	return json.NewDecoder(r.Body).Decode(value)
}

func setError(w http.ResponseWriter, err apiError) {
	w.WriteHeader(err.Status)
	w.Write([]byte(err.Msg))
}
