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
	//"time"
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

func handleQuery(w http.ResponseWriter, r *http.Request) {
	//r.URL.Query()
	if val, ok := r.URL.Query()["recent_payments"]; ok {
		i, _ := strconv.Atoi(val[0])
		handleRows(i, w, r)
	}
	if val, ok := r.URL.Query()["pk"]; ok {
		i, _ := strconv.Atoi(val[0])
		handleId(i, w, r)
	}

}

func handleRows(rowCount int, w http.ResponseWriter, r *http.Request) {
	log.Println("in handleRows")

	rows, err := db.Query(fmt.Sprintf("SELECT * FROM test ORDER BY pk LIMIT %d", rowCount))
	defer rows.Close()

	if err != nil {
		log.Println(err)
	}


	buffer := dumpTableToHTML(rows, "Merchants")

	t, _ := template.ParseFiles("dumper_results.html")
	t.Execute(w, template.HTML(buffer.String()))
}


func handleId(id int, w http.ResponseWriter, r *http.Request) {

	row, _ := db.Query(fmt.Sprintf("SELECT pk, val FROM test where pk = %d", id))
	defer row.Close()

	buffer := dump2ColTableToHTML(row, "test")


	t, _ := template.ParseFiles("dumper_results.html")
	t.Execute(w, template.HTML(buffer.String()))
}

func parseRowsToColsAndData(rows *sql.Rows) (columnNames []string, tableData[][]string){
	log.Println("in parseRows")
	// Get column names
	columnNames, _ = rows.Columns()


	// Make a slice for the values
	values := make([]sql.RawBytes, len(columnNames))

	// rows.Scan wants '[]interface{}' as an argument, so we must copy the
	// references into such a slice
	// See http://code.google.com/p/go-wiki/wiki/InterfaceSlice for details
	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	// Fetch rows
	for rows.Next() {
		log.Println("looping over rows ")

		// get RawBytes from data
		_ = rows.Scan(scanArgs...)

		// create the slice we'll use to return the data
		rowData := make([]string, len(columnNames))

		// Now do something with the data.
		// Here we just print each column as a string.
		var value string
		for i, col := range values {
			// Here we can check if the value is nil (NULL value)
			if col == nil {
				value = "NULL"
			} else {
				value = string(col)
			}
			rowData[i] = value
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
		buffer.WriteString(fmt.Sprintf("<td>%s</td>", getFieldHTML(columnNames[i],row[i])))
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
			buffer.WriteString(getFieldHTML(columnNames[i],row[i]))
			buffer.WriteString("</td>")
		}
		buffer.WriteString("</tr>")
	}
	buffer.WriteString("</table>")

	return buffer

}

var link_map = map[string]struct{}{
	"pk" : {},
}


func getFieldHTML(colName string, colData string) (string) {
	// is it a link?
	if _, ok := link_map[colName]; ok {
		return getLinkHTML(colName, colData)
	}

	return colData
}

func getLinkHTML(colName string, colData string) (string) {
	//$link = $this->script_name . "?$field_name=$field_data";
	//$display = $display ? $display : $field_data;
	return fmt.Sprintf("<a href=\"?%s=%s\">%s</a>", colName, colData, colData)
}

func main() {
	var err error
	db, err = sql.Open("postgres",
		"user= dbname= sslmode=disable")
	defer db.Close()

	if err != nil {
		fmt.Printf("got an error: %v\n", err)
	}

	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}


