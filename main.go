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

type ApiResponse struct {
	Hostname string `json:"hostname"`
}

type ProductResponse struct {
	Product     datahandling.Product `json:"product"`
	ApiResponse ApiResponse          `json:"apiresponse"`
}

type ProductsResponse struct {
	Products    []datahandling.Product `json:"products"`
	ApiResponse ApiResponse            `json:"apiresponse"`
}

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
	http.HandleFunc("/getProductsBySearchValues", handleGetProductsBySearchValues(ctx, db, queries))
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
			setError(w, ErrReadJSON, err.Error())
			return
		}

		name := nameJSON["name"].(string)
		price := nameJSON["price"].(float64)
		catId := nameJSON["categoryId"].(float64)
		details := nameJSON["details"].(string)

		product := datahandling.AddProductParams{
			Name:       name,
			Price:      price,
			CategoryID: int32(int(catId)),
			Details:    details,
		}

		resp, err := http.Get("http://category-service:8081/getCategory?id=" + strconv.Itoa(int(catId)))

		if err != nil {
			setError(w, ErrRequestFailed, err.Error())
			return
		}

		if resp.StatusCode == http.StatusOK {
			err = queries.AddProduct(ctx, product)
			if err != nil {
				setError(w, ErrQueryDatabase, err.Error())
				return
			}
		}

		hostname, err := os.Hostname()
		if err != nil {
			setError(w, ErrHostname, err.Error())
			return
		}
		apiResponse := ApiResponse{Hostname: hostname}

		err = writeJSONResponse(w, http.StatusOK, apiResponse)
		if err != nil {
			setError(w, ErrWriteJSON, err.Error())
		}
	}
}

func handleGetProductById(ctx context.Context, queries *datahandling.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")

		if idStr == "" {
			setError(w, ErrIDNotSet, "")
			return
		}

		id, err := strconv.Atoi(idStr)
		if err != nil {
			setError(w, ErrStrToInt, err.Error())
			return
		}

		product, err := queries.GetProduct(ctx, int32(id))
		if err != nil {
			setError(w, ErrQueryDatabase, err.Error())
			return
		}

		hostname, err := os.Hostname()
		if err != nil {
			setError(w, ErrHostname, err.Error())
			return
		}
		apiResponse := ApiResponse{Hostname: hostname}

		prodResponse := ProductResponse{
			Product:     product,
			ApiResponse: apiResponse,
		}

		err = writeJSONResponse(w, http.StatusOK, prodResponse)
		if err != nil {
			setError(w, ErrWriteJSON, err.Error())
			return
		}
	}
}

func handleGetProducts(ctx context.Context, queries *datahandling.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		products, apiErr := queryAllProducts(ctx, queries)
		if apiErr != nil {
			setError(w, *apiErr, "")
			return
		}

		hostname, err := os.Hostname()
		if err != nil {
			setError(w, ErrHostname, err.Error())
			return
		}
		apiResponse := ApiResponse{Hostname: hostname}

		prodResponse := ProductsResponse{
			Products:    products,
			ApiResponse: apiResponse,
		}

		err = writeJSONResponse(w, http.StatusOK, prodResponse)
		if err != nil {
			setError(w, ErrWriteJSON, err.Error())
		}
	}
}

func handleGetProductByName(ctx context.Context, queries *datahandling.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")

		products := make([]datahandling.Product, 0)
		queriedProducts, err := queries.GetProductByName(ctx, name)
		if err != nil {
			setError(w, ErrQueryDatabase, err.Error())
			return
		}
		products = append(products, queriedProducts...)

		hostname, err := os.Hostname()
		if err != nil {
			setError(w, ErrHostname, err.Error())
			return
		}
		apiResponse := ApiResponse{Hostname: hostname}

		prodResponse := ProductsResponse{
			Products:    products,
			ApiResponse: apiResponse,
		}

		err = writeJSONResponse(w, http.StatusOK, prodResponse)
		if err != nil {
			setError(w, ErrWriteJSON, err.Error())
		}
	}
}

func handleDelProductById(ctx context.Context, queries *datahandling.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			setError(w, ErrMethodNotAllowed, "")
			return
		}

		idStr := r.URL.Query().Get("id")

		if idStr == "" {
			setError(w, ErrIDNotSet, "")
			return
		}

		id, err := strconv.Atoi(idStr)
		if err != nil {
			setError(w, ErrStrToInt, "")
			return
		}

		if id < 0 {
			setError(w, ErrIDNegative, "")
			return
		}

		err = queries.DelProduct(ctx, int32(id))
		if err != nil {
			setError(w, ErrQueryDatabase, err.Error())
			return
		}

		hostname, err := os.Hostname()
		if err != nil {
			setError(w, ErrHostname, err.Error())
			return
		}
		apiResponse := ApiResponse{Hostname: hostname}

		err = writeJSONResponse(w, http.StatusOK, apiResponse)
		if err != nil {
			setError(w, ErrWriteJSON, err.Error())
		}
	}
}

func handleDelProductByCategoryId(ctx context.Context, queries *datahandling.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			setError(w, ErrStrToInt, err.Error())
			return
		}

		err = queries.DelProductsByCategory(ctx, int32(id))
		if err != nil {
			setError(w, ErrQueryDatabase, err.Error())
			return
		}
		hostname, err := os.Hostname()
		if err != nil {
			setError(w, ErrHostname, err.Error())
			return
		}
		apiResponse := ApiResponse{Hostname: hostname}

		err = writeJSONResponse(w, http.StatusOK, apiResponse)
		if err != nil {
			setError(w, ErrWriteJSON, err.Error())
		}
	}
}
func handleGetProductsBySearchValues(ctx context.Context, db *sql.DB, queries *datahandling.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		details := query.Get("details")
		minPriceStr := query.Get("minPrice")
		maxPriceStr := query.Get("maxPrice")

		//return all Products
		if details == "" && minPriceStr == "" && maxPriceStr == "" {
			productMap, apiErr := queryAllProducts(ctx, queries)
			if apiErr != nil {
				setError(w, *apiErr, "")
				return
			}

			err := writeJSONResponse(w, http.StatusOK, productMap)
			if err != nil {
				setError(w, ErrWriteJSON, err.Error())
				return
			}
			return
		}

		products, apiErr := searchProducts(minPriceStr, maxPriceStr, details, db)
		if apiErr != nil {
			setError(w, *apiErr, "")
			return
		}

		hostname, err := os.Hostname()
		if err != nil {
			setError(w, ErrHostname, err.Error())
			return
		}
		apiResponse := ApiResponse{Hostname: hostname}

		prodResponse := ProductsResponse{
			Products:    products,
			ApiResponse: apiResponse,
		}

		err = writeJSONResponse(w, http.StatusOK, prodResponse)
		if err != nil {
			setError(w, ErrWriteJSON, err.Error())
		}
	}
}

func queryAllProducts(ctx context.Context, queries *datahandling.Queries) ([]datahandling.Product, *apiError) {
	products := make([]datahandling.Product, 0)

	queriedProducts, err := queries.GetProducts(ctx)
	if err != nil {
		return nil, &ErrQueryDatabase
	}

	products = append(products, queriedProducts...)

	return products, nil
}

func searchProducts(minPriceStr, maxPriceStr, details string, db *sql.DB) ([]datahandling.Product, *apiError) {
	sql := "SELECT * FROM product WHERE 1=1"
	args := []any{}

	if minPriceStr != "" {
		minPrice, apiErr := parsePrice(minPriceStr)
		if apiErr != nil {
			return nil, apiErr
		}
		sql += " AND price >= ?"
		args = append(args, minPrice)
	}

	if maxPriceStr != "" {
		maxPrice, apiErr := parsePrice(maxPriceStr)
		if apiErr != nil {
			return nil, apiErr
		}
		sql += " AND price <= ?"
		args = append(args, maxPrice)
	}

	if details != "" {
		sql += " AND details LIKE ?"
		args = append(args, "%"+details+"%")
	}

	sql += ";"
	stmt, err := db.Prepare(sql)
	if err != nil {
		return nil, &ErrPrepareQuery
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, &ErrQueryDatabase
	}
	defer rows.Close()

	products := make([]datahandling.Product, 0)
	queriedProducts, apiErr := scanProductRows(rows)
	if apiErr != nil {
		return nil, apiErr
	}
	products = append(products, queriedProducts...)

	return products, nil
}

func parsePrice(priceStr string) (*float64, *apiError) {
	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		return nil, &ErrStrToFloat
	}
	if price < 0 {
		return nil, &ErrNegativePrice
	}
	return &price, nil
}

func scanProductRows(rows *sql.Rows) ([]datahandling.Product, *apiError) {
	var products []datahandling.Product

	for rows.Next() {
		var p datahandling.Product
		err := rows.Scan(&p.ID, &p.Details, &p.Name, &p.Price, &p.CategoryID)
		if err != nil {
			return nil, &ErrRowScan
		}
		products = append(products, p)
	}

	return products, nil
}

func writeJSONResponse(w http.ResponseWriter, status int, value any) error {
	w.WriteHeader(status)
	w.Header().Add("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(value)
}

func readJSON(r *http.Request, value *map[string]any) error {
	return json.NewDecoder(r.Body).Decode(value)
}

func setError(w http.ResponseWriter, err apiError, returnedErrMsg string) {
	fmt.Println(err.Msg)
	fmt.Println(returnedErrMsg)
	w.WriteHeader(err.Status)
	w.Write([]byte(err.Msg))
}
