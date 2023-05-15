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
	http.HandleFunc("/delProductsByCategoryId", handleDelProductByCategoryId(ctx, queries))

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
		catId := nameJSON["categoryId"].(float64)
		details := nameJSON["details"].(string)

		// exists, err := queries.ProductExists(ctx, name)

		// if exists {
		// 	fmt.Println("Already Exists!")
		// 	setError(w, ErrAlreadyExists)
		// 	return
		// }

		// if err != nil {
		// 	fmt.Println(err.Error())
		// 	setError(w, ErrQueryDatabase)
		// 	return
		// }

		product := datahandling.AddProductParams{
			Name:       name,
			Price:      price,
			CategoryID: int32(int(catId)),
			Details:    details,
		}

		fmt.Println(product)

		resp, err := http.Get("http://category-service:8081/getCategory?id=" + strconv.Itoa(int(catId)))

		if err != nil {
			fmt.Println(err.Error())
			setError(w, ErrRequestFailed)
			return
		}

		if resp.StatusCode == http.StatusOK {
			_, err = queries.AddProduct(ctx, product)
			if err != nil {
				fmt.Println(err.Error())
				setError(w, ErrQueryDatabase)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
	}
}

func handleGetProductById(ctx context.Context, queries *datahandling.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		id, err := strconv.Atoi(idStr)

		if err != nil {
			setError(w, ErrStrToInt)
			return
		}

		product, err := queries.GetProduct(ctx, int32(id))
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
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}
}

func handleGetProductByName(ctx context.Context, queries *datahandling.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")

		products, err := queries.GetProductByName(ctx, name)
		if err != nil {
			setError(w, ErrQueryDatabase)
			return
		}

		fmt.Println(products)

		productMap := map[string][]datahandling.Product{
			"products": products,
		}

		err = writeJSON(w, http.StatusOK, productMap)
		if err != nil {
			setError(w, ErrWriteJSON)
		}
	}
}

func handleGetProductsBySearchValues(ctx context.Context, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		details := query.Get("details")
		minPriceStr := query.Get("minPrice")
		maxPriceStr := query.Get("maxPrice")

		if details == "" && minPriceStr == "" && maxPriceStr == "" {
			setError(w, ErrNoSearchParamSet)
			return
		}

		sql := "SELECT * FROM products WHERE 1=1"
		args := []any{}

		if minPriceStr != "" {
			minPrice, err := strconv.Atoi(minPriceStr)
			if err != nil {
				setError(w, ErrStrToInt)
				return
			}
			if minPrice < 0 {
				setError(w, ErrNegativePrice)
				return
			}
			sql += "AND price >= ?"
			args = append(args, minPrice)
		}

		if maxPriceStr != "" {
			maxPrice, err := strconv.Atoi(minPriceStr)
			if err != nil {
				setError(w, ErrStrToInt)
				return
			}
			if maxPrice < 0 {
				setError(w, ErrNegativePrice)
				return
			}
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
		fmt.Println("Trying to delete!")
		if r.Method != http.MethodDelete {
			setError(w, ErrMethodNotAllowed)
			return
		}

		idStr := r.URL.Query().Get("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			setError(w, ErrStrToInt)
			return
		}

		err = queries.DelProduct(ctx, int32(id))
		if err != nil {
			setError(w, ErrQueryDatabase)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func handleDelProductByCategoryId(ctx context.Context, queries *datahandling.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			setError(w, ErrStrToInt)
			return
		}

		err = queries.DelProduct(ctx, int32(id))
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
	err := json.NewEncoder(w).Encode(value)
	fmt.Println(value)
	return err
}

func readJSON(r *http.Request, value *map[string]any) error {
	return json.NewDecoder(r.Body).Decode(value)
}

func setError(w http.ResponseWriter, err apiError) {
	fmt.Println(err.Msg)
	w.WriteHeader(err.Status)
	w.Write([]byte(err.Msg))
}
