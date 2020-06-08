package main

import (
	"encoding/json"
	"io"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jinzhu/gorm/dialects/mysql"

	// convert the String variable into an Integer before we run the SQL queries
	"strconv" 

	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"

	// connect the API server to the front-end page
	"github.com/rs/cors" 
)

// TodoItemModel - id - primary_key, Description - display in front-end UI, Completed - if the TodoItem is done or not
type TodoItemModel struct {
	ID          int `gorm:"primary_key"`
	Description string
	Completed   bool
}

// initialize MySQL connection to database using GORM
var db, _ = gorm.Open("mysql", "root:root@/todolist?charset=utf8&parseTime=True&loc=Local")

// Healthz - created a Healthz function
//that’ll respond {"alive": true} to the client every time it’s invoked.
func Healthz(w http.ResponseWriter, r *http.Request) {
	log.Info("API Health is OK")
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"alive": true}`)
}

// set our init function to set up our logrus logger settings.
// In Golang, init() will be executed when the program first begins
func init() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetReportCaller(true)
}

// CreateItem -
func CreateItem(w http.ResponseWriter, r *http.Request) {
	description := r.FormValue("description")                                                            // obtain the value from the POST operation with r.FormValue("parameter")
	log.WithFields(log.Fields{"description": description}).Info("Add new TodoItem. Saving to database.") // use the value as the description to insert into our database
	todo := &TodoItemModel{Description: description, Completed: false}                                   // create the todo object and persist it in our database
	db.Create(&todo)

	// query the database and return the query result to the client to make sure the operation is success
	result := db.Last(&todo)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result.Value)
}

// UpdateItem -
func UpdateItem(w http.ResponseWriter, r *http.Request) {
	// Get URL parameter from mux
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	// Test if the TodoItem exist in DB
	err := GetItemByID(id)
	if err == false {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"updated": false, "error": "Record Not Found"}`)
	} else {
		completed, _ := strconv.ParseBool(r.FormValue("completed"))
		log.WithFields(log.Fields{"Id": id, "Completed": completed}).Info("Updating TodoItem")
		todo := &TodoItemModel{}
		db.First(&todo, id)
		todo.Completed = completed
		db.Save(&todo)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"updated": true}`)
	}
}

// DeleteItem -
func DeleteItem(w http.ResponseWriter, r *http.Request) {
	// Get URL parameter from mux
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	// Test if the TodoItem exist in DB
	err := GetItemByID(id)
	if err == false {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"deleted": false, "error": "Record Not Found"}`)
	} else {
		log.WithFields(log.Fields{"Id": id}).Info("Deleting TodoItem")
		todo := &TodoItemModel{}
		db.First(&todo, id)
		db.Delete(&todo)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"deleted": true}`)
	}
}

// GetItemByID -
func GetItemByID(ID int) bool {
	todo := &TodoItemModel{}
	result := db.First(&todo, ID)
	if result.Error != nil {
		log.Warn("TodoItem not found in database")
		return false
	}
	return true
}

// GetCompletedItems - execute a SQL SELECT query and encode it into JSON before returning it to the client
func GetCompletedItems(w http.ResponseWriter, r *http.Request) {
	log.Info("Get completed TodoItems")
	completedTodoItems := GetTodoItems(true)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(completedTodoItems)
}

// GetIncompleteItems - execute a SQL SELECT query and encode it into JSON before returning it to the client
func GetIncompleteItems(w http.ResponseWriter, r *http.Request) {
	log.Info("Get Incomplete TodoItems")
	IncompleteTodoItems := GetTodoItems(false)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(IncompleteTodoItems)
}

// GetTodoItems -
func GetTodoItems(completed bool) interface{} {
	var todos []TodoItemModel
	TodoItems := db.Where("completed = ?", completed).Find(&todos).Value
	return TodoItems
}

// initializing our gorilla/mux HTTP router
// route /healthz HTTP GET requests to the Health() function
// important to set the Content-Type: application/json
// so the client software understands the response
func main() {
	// close database connection main() function is returned
	defer db.Close()

	// running automigration against our MySQL database immediately after starting our API server
	db.Debug().DropTableIfExists(&TodoItemModel{})
	db.Debug().AutoMigrate(&TodoItemModel{})

	log.Info("Starting Todolist API server")
	router := mux.NewRouter()
	router.HandleFunc("/healthz", Healthz).Methods("GET")
	router.HandleFunc("/todo-completed", GetCompletedItems).Methods("GET")
	router.HandleFunc("/todo-incomplete", GetIncompleteItems).Methods("GET")
	router.HandleFunc("/todo", CreateItem).Methods("POST")
	router.HandleFunc("/todo/{id}", UpdateItem).Methods("POST")
	router.HandleFunc("/todo/{id}", DeleteItem).Methods("DELETE")
	http.ListenAndServe(":8000", router)

	// wrapping the CORS handler around our existing application
	handler := cors.New(cors.Options{
	AllowedMethods: []string{"GET", "POST", "DELETE", "PATCH", "OPTIONS"},
 	}).Handler(router)
	http.ListenAndServe("localhost:8080", handler)
}
