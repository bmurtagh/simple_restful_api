package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"
	//These packages need to be downloaded
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

type Room struct {
	ID                 int    `json:"id"`
	Name               string `json:"name"`
	Size               int    `json:"size"`
	WindowCount        int    `json:"windowCount"`
	WallDecorationType string `json:"wallDecorationType"`
	Floor              int    `json:"floor"`
	Doors              []Door `json:"doors"`
}

type Door struct {
	Destination string `json:"destination"`
}

var house []Room

var db *sql.DB

func GetHouseInfo(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(house)
}

func GetRoomInfo(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	unescaped, err := url.QueryUnescape(params["roomName"])
	checkErr(err)
	for _, item := range house {
		lowerName := strings.ToLower(item.Name)
		if lowerName == unescaped {
			json.NewEncoder(w).Encode(item)
			return
		}
	}
	json.NewEncoder(w).Encode(&Room{})
}

func CreateRoom(w http.ResponseWriter, r *http.Request) {
	var room Room
	err := json.NewDecoder(r.Body).Decode(&room)
	checkErr(err)
	if room.Name == "" {
		room.Name = "no-name"
	}
	var id int
	id, err = addRow(room)
	checkErr(err)
	room.ID = id
	house = append(house, room)
	json.NewEncoder(w).Encode(house)
}

func DeleteRoom(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	unescaped, err := url.QueryUnescape(params["roomName"])
	checkErr(err)
	for i, item := range house {
		lowerName := strings.ToLower(item.Name)
		if lowerName == unescaped {
			err = removeRow(house[i].ID)
			checkErr(err)
			house = append(house[:i], house[i+1:]...)
			break
		}
	}
	json.NewEncoder(w).Encode(house)
}

func main() {
	// Open a database connection
	var err error
	db, err = sql.Open("sqlite3", "./db/house.db")
	checkErr(err)
	// Check if reachable
	if err = db.Ping(); err != nil {
		log.Fatal("Database is unreachable:", err)
	}
	// Populate variables with data
	err = populateVars()
	checkErr(err)

	// Route the resource requests
	router := mux.NewRouter()
	router.HandleFunc("/house", GetHouseInfo).Methods("GET")
	router.HandleFunc("/house/{roomName}", GetRoomInfo).Methods("GET")
	router.HandleFunc("/house/new", CreateRoom).Methods("POST")
	router.HandleFunc("/house/{roomName}", DeleteRoom).Methods("DELETE")
	log.Fatal(http.ListenAndServe(":8080", router))

}

func checkErr(err error) {
	if err != nil {
		log.Println(err)
	}
}

func addRow(row Room) (int, error) {
	stmt, err := db.Prepare("INSERT INTO Rooms (Name, Size, WindowCount, WallDecorationType, Floor) VALUES(?, ?, ?, ?, ?)")
	var res sql.Result
	res, err = stmt.Exec(row.Name, row.Size, row.WindowCount, row.WallDecorationType, row.Floor)
	var id int64
	id, err = res.LastInsertId()
	for i := range row.Doors {
		stmt, err = db.Prepare("INSERT INTO Doors (RoomID, Destination) VALUES(?, ?)")
		_, err = stmt.Exec(id, row.Doors[i].Destination)
	}
	return int(id), err
}

func removeRow(id int) error {
	stmt, err := db.Prepare("DELETE FROM Rooms WHERE RoomID IS ?")
	_, err = stmt.Exec(id)
	stmt, err = db.Prepare("DELETE FROM Doors WHERE RoomID IS ?")
	_, err = stmt.Exec(id)
	return err
}

func populateVars() error {
	rows, err := db.Query("SELECT * FROM Rooms")
	checkErr(err)
	i := 0
	for rows.Next() {
		house = append(house, Room{})
		err = rows.Scan(&house[i].ID, &house[i].Name, &house[i].Size, &house[i].WindowCount, &house[i].WallDecorationType, &house[i].Floor)
		if err != nil {
			return err
		}
		i++
	}
	rows, err = db.Query("SELECT * FROM Doors")
	checkErr(err)
	for rows.Next() {
		var id int
		var dest string
		err = rows.Scan(&id, &dest)
		if err != nil {
			return err
		}
		for i := range house {
			if house[i].ID == id {
				house[i].Doors = append(house[i].Doors, Door{Destination: dest})
			}
		}
	}
	rows.Close()
	return nil
}
