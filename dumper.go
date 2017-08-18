package main

import (
	"html/template"
	"net/http"
	"strconv"
	"fmt"
	"database/sql"
	_ "github.com/lib/pq"
	"log"
	"bytes"
	"strings"
)

var db *sql.DB


func handler(w http.ResponseWriter, r *http.Request) {
	if len(r.URL.Query()) > 0 {
		handleQuery(w,r)
	} else {
		home(w,r)
	}
}

func home(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("dumper.html")
	t.Execute(w, nil)
}

func getRowCountFromQuery(r *http.Request, key string) (int, bool) {
	val, ok := r.URL.Query()[key]

	if !ok {
		return 0, ok
	} else {
		rows, err := strconv.Atoi(val[0])

		if err != nil {
			log.Println(err)
			return 0, false
		}

		if rows > 250 {
			return 250, ok
		} else {
			return rows, ok
		}
	}
}

func handleQuery(w http.ResponseWriter, r *http.Request) {
	var buffer bytes.Buffer

	log.Println("Query: ", r.URL.Query())

	// Load the query parameters into an array
	var parameters  []string
	for key, _ := range r.URL.Query() {
		parameters = append(parameters, key)
	}

	log.Println("Parameters: ", parameters)

	if len(parameters) == 0 {
		home(w,r)
		return
	}

	if rowCount, ok := getRowCountFromQuery(r, "recent_books"); ok {
		buffer = handleRows(rowCount, "book" )
	} else if strings.HasSuffix(parameters[0], "_id") {
		// Catchall for id's that don't have special handling.
		// This check must go after any id's with special handling
		buffer = handleId(parameters[0], r.URL.Query()[parameters[0]][0])
	} else {
		home(w,r)
		return
	}

	t, _ := template.ParseFiles("dumper_results.html")
	t.Execute(w, template.HTML(buffer.String()))

}

func handleRows(rowCount int, tableName string) bytes.Buffer {
	log.Printf("in handleRows. rowCount: %d tableName: %s", rowCount, tableName)

	rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s ORDER BY created_at DESC LIMIT %d", tableName, rowCount))
	defer rows.Close()

	if err != nil {
		log.Println(err)
	}


	return dumpTableToHTML(rows, tableName)
}

// Maps id column names to the tables where the data lives
var idname_to_table_map = map[string]string {
	"book_id" : "book",
	"author_id" : "author",
}


func handleId(id_name string, id string) bytes.Buffer {
	log.Println("in handleId")

	var tableName string

	tableName, ok := idname_to_table_map[id_name]
	if !ok {
		var buffer bytes.Buffer
		buffer.WriteString(fmt.Sprintf("Don't know how to handle_id [%s]", id_name));
		return buffer
	}

	row, err := db.Query(fmt.Sprintf("SELECT * FROM %s where %s = '%s'", tableName, id_name, id))
	defer row.Close()

	if err != nil {
		log.Println(err)
		var buffer bytes.Buffer
		buffer.WriteString(fmt.Sprintf("I had a problem querying [%s]", tableName));
		return buffer
	}

	return dump2ColTableToHTML(row, tableName)
}

// maps column names that don't follow the standard FK naming convention to their linked table
// If it's a column name that looks like a foreign key, but isn't, like tax_id it will map to a nil
var non_links = map[string]string {
	"not_really_an_id" : "",
}

// column names with special handling
type fn func (string) string
var special_map = map[string] fn {
	"ssn" : getRedactedHTML,
}


func getFieldHTML(tableName string, colName string, colData string) (string) {
	// it is a link?
	if _, isNonLink := non_links[colName]; strings.HasSuffix(colName, "_id") && !isNonLink {
		return getLinkHTML(colName, colData)
	}

	if fn, ok := special_map[colName]; ok {
		return fn(colData)
	}

	return colData
}

func getRedactedHTML(colData string) string {
	if colData != "" {
		return "<span style=\"color:red;\">REDACTED</SPAN>";
	} else {
		return ""
	}
}

func getLinkHTML(colName string, colData string) (string) {
	return fmt.Sprintf("<a href=\"?%s=%s\">%s</a>", colName, colData, colData)
}

func parseRowsToColsAndData(rows *sql.Rows) (columnNames []string, tableData[][]string){
	log.Println("in parseRows")
	// Get column names
	columnNames, _ = rows.Columns()

	columns := make([]interface{}, len(columnNames))
	columnPointers := make([]interface{}, len(columnNames))
	for i := 0; i < len(columnNames); i++ {
		columnPointers[i] = &columns[i]
	}

	// Fetch rows
	for rows.Next() {
		log.Println("looping over rows ")

		if err := rows.Scan(columnPointers...); err != nil {
			log.Fatalln(err)
		}

		// create the slice we'll use to return the data
		rowData := make([]string, len(columnNames))


		for i, value := range columns {
			switch value.(type) {
			case []uint8:
				sb := value.([]uint8)
				rowData[i] = string(sb)
			default:
				rowData[i] = fmt.Sprintf("%v", value)
			}
		}

		tableData = append(tableData, rowData)
	}

	return
}


func dump2ColTableToHTML(rows *sql.Rows, tableName string) bytes.Buffer {

	columnNames, tableData := parseRowsToColsAndData(rows)
	row := tableData[0]

	var buffer bytes.Buffer

	// Top of the page
	buffer.WriteString(fmt.Sprintf("<h2>%s</h2>", tableName))

	// Begin the table
	buffer.WriteString("<table border=1>")

	// One row per field
	for i := 0; i < len(columnNames); i++ {
		buffer.WriteString("<tr>")
		// Write the colname as a header
		buffer.WriteString(fmt.Sprintf("<th>%s</th>", columnNames[i]))
		// Write the val as a regular cell
		buffer.WriteString(fmt.Sprintf("<td>%s</td>", getFieldHTML(tableName, columnNames[i], row[i])))
		buffer.WriteString("</tr>")
	}
	buffer.WriteString("</table>")

	return buffer

}

func dumpTableToHTML(rows *sql.Rows, tableName string) bytes.Buffer {

	columnNames, tableData := parseRowsToColsAndData(rows)
	var buffer bytes.Buffer

	// Top of the page
	buffer.WriteString(fmt.Sprintf("<h2>%s</h2>", tableName))

	// Begin the table
	buffer.WriteString("<table border=1>")

	// Write the header
	buffer.WriteString("<tr>")
	for _, name := range columnNames {
		buffer.WriteString(fmt.Sprintf("<th>%s</th>", name))
	}
	buffer.WriteString("</tr>")

	// Write each row
	for _, row := range tableData {
		buffer.WriteString("<tr>")
		// Write each column
		for i := 0; i < len(columnNames); i++ {
			buffer.WriteString("<td>")
			buffer.WriteString(getFieldHTML(tableName,columnNames[i],row[i]))
			buffer.WriteString("</td>")
		}
		buffer.WriteString("</tr>")
	}
	buffer.WriteString("</table>")

	return buffer

}

func main() {
	var err error
	db, err = sql.Open("postgres",
		"user=billmassie dbname=billmassie sslmode=disable")
	defer db.Close()

	if err != nil {
		fmt.Printf("got an error: %v\n", err)
	}

	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}


