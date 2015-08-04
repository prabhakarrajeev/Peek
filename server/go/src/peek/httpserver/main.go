package main

import (
	"code.google.com/p/go-uuid/uuid"
	"encoding/json"
	"fmt"
	"github.com/fzzy/radix/redis"
	"net/http"
	"os"
	"peek/service"
	"strconv"
	"time"
)

type Boat struct {
	Id       string
	Capacity int64
	Name     string
}

type Timeslot struct {
	Id             string
	Start_time     int64
	Duration       int64
	Availability   int64
	Customer_count int64
	Boats          []Boat
}

func max(a, b int64) int64 {
	if a < b {
		return b
	}
	return a
}

func TimeslotHandler(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()
	start_time, _ := strconv.ParseInt(r.FormValue("timeslot[start_time]"), 10, 0)
	dur, _ := strconv.ParseInt(r.FormValue("timeslot[duration]"), 10, 0)
	date := r.FormValue("date")
	var timeslot []byte

	redisClient, _ := redis.Dial("tcp", "127.0.0.1:6379")

	if len(date) == 0 {

		t1 := time.Unix(start_time, 0)
		dateKey := fmt.Sprintf("%d-%02d-%02d", t1.Year(), t1.Month(), t1.Day())

		v := &Timeslot{
			Id:             uuid.New(),
			Start_time:     start_time,
			Duration:       dur,
			Availability:   0,
			Customer_count: 0,
			Boats:          []Boat{},
		}
		timeslot, _ = json.Marshal(v)

		redisClient.Cmd("MULTI")
		redisClient.Cmd("SET", "ts:"+v.Id, timeslot)
		redisClient.Cmd("SADD", dateKey, "ts:"+v.Id)
		redisClient.Cmd("EXEC")
	} else {
		var ts []interface{}
		r := redisClient.Cmd("SMEMBERS", date)

		for i := range r.Elems {
			elemStr, _ := r.Elems[i].Str()
			data, _ := redisClient.Cmd("GET", elemStr).Bytes()

			var timestamp interface{}
			json.Unmarshal(data, &timestamp)

			bs := timestamp.(map[string]interface{})["Boats"]
			for i = 0; i < len(bs.([]interface{})); i++ {
				Id := (bs.([]interface{})[i]).(map[string]interface{})["Id"]
				delete((bs.([]interface{})[i]).(map[string]interface{}), "Id")
				delete((bs.([]interface{})[i]).(map[string]interface{}), "Name")
				delete((bs.([]interface{})[i]).(map[string]interface{}), "Capacity")
				bs.([]interface{})[i] = Id
			}
			ts = append(ts, timestamp)
		}
		timeslot, _ = json.Marshal(ts)
	}

	redisClient.Close()
	status := 200
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(timeslot))
}

func BoatsHandler(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()
	capacity, _ := strconv.ParseInt(r.FormValue("boat[capacity]"), 10, 0)
	name := r.FormValue("boat[name]")
	var boats []byte

	redisClient, _ := redis.Dial("tcp", "127.0.0.1:6379")

	if len(name) == 0 {
		var bs []interface{}
		r := redisClient.Cmd("SMEMBERS", "boats")

		for i := range r.Elems {
			elemStr, _ := r.Elems[i].Str()
			data, _ := redisClient.Cmd("GET", elemStr).Bytes()

			var boat interface{}
			json.Unmarshal(data, &boat)
			bs = append(bs, boat)
		}
		boats, _ = json.Marshal(bs)

	} else {
		v := &Boat{
			Id:       uuid.New(),
			Capacity: capacity,
			Name:     name,
		}
		boats, _ = json.Marshal(v)
		redisClient.Cmd("SET", "boat:"+v.Id, boats)
		redisClient.Cmd("SADD", "boats", "boat:"+v.Id)
	}

	redisClient.Close()
	status := 200
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(boats))
}

func AssignmentsHandler(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()
	timeslot_id := r.FormValue("assignment[timeslot_id]")
	boat_id := r.FormValue("assignment[boat_id]")

	redisClient, _ := redis.Dial("tcp", "127.0.0.1:6379")

	data_ts, _ := redisClient.Cmd("GET", "ts:"+timeslot_id).Bytes()
	data_boat, _ := redisClient.Cmd("GET", "boat:"+boat_id).Bytes()

	var boat Boat
	json.Unmarshal(data_boat, &boat)

	var timestamp Timeslot
	json.Unmarshal(data_ts, &timestamp)

	timestamp.Boats = append(timestamp.Boats, boat)
	timestamp.Availability = max(timestamp.Availability, boat.Capacity)

	ts, _ := json.Marshal(timestamp)
	redisClient.Cmd("SET", "ts:"+timeslot_id, ts)
	redisClient.Cmd("SADD", "asmt:"+boat_id, "ts:"+timeslot_id)

	redisClient.Close()
}

func BookingHandler(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()
	timeslot_id := r.FormValue("booking[timeslot_id]")
	booking_size, _ := strconv.ParseInt(r.FormValue("booking[size]"), 10, 0)

	redisClient, _ := redis.Dial("tcp", "127.0.0.1:6379")

	for retries := 10; retries >= 0; retries-- {

		data_ts, _ := redisClient.Cmd("GET", "ts:"+timeslot_id).Bytes()
		var timeslot Timeslot
		json.Unmarshal(data_ts, &timeslot)

		if booking_size > timeslot.Availability {
			fmt.Println("Booking : No Available slots")
			return
		}

		//select smallest boat big enough for this booking
		var index int = -1
		var least_size int64 = int64(^uint(0) >> 1)
		var avail int64 = 0
		var associated_ts []string
		for i, val := range timeslot.Boats {
			if val.Capacity >= booking_size {
				if val.Capacity <= least_size {
					index = i
					least_size = val.Capacity
				}
			}

			//Watch out for any changes to timeslots associated with the boats after a read
			r := redisClient.Cmd("SMEMBERS", "asmt:"+val.Id)
			for i := range r.Elems {
				timeStamp, _ := r.Elems[i].Str()
				redisClient.Cmd("WATCH", timeStamp)
				associated_ts = append(associated_ts, timeStamp)
			}
		}

		//Update the requested timeslot
		timeslot.Boats[index].Capacity -= booking_size
		timeslot.Customer_count += booking_size
		for _, val := range timeslot.Boats {
			avail = max(avail, val.Capacity)
		}
		timeslot.Availability = avail
		booking_ts, _ := json.Marshal(timeslot)

		//if there are other trips overlaping with this timeslot, handle them
		booking_boatId := timeslot.Boats[index].Id
		var assoc_ts_map = make(map[string][]byte)
		for _, tslot := range associated_ts {

			if tslot != "ts:"+timeslot_id {

				dta_ts, _ := redisClient.Cmd("GET", tslot).Bytes()
				var ts Timeslot
				json.Unmarshal(dta_ts, &ts)

				if (ts.Start_time < timeslot.Start_time && ts.Start_time+ts.Duration*60 > timeslot.Start_time) ||
					(ts.Start_time > timeslot.Start_time && timeslot.Start_time+timeslot.Duration*60 > ts.Start_time) {

					var ts_avail int64 = 0
					for idx, boat := range ts.Boats {
						if boat.Id == booking_boatId {
							ts.Boats[idx].Capacity = 0
						}
						ts_avail = max(ts_avail, ts.Boats[idx].Capacity)
					}
					ts.Availability = ts_avail
					bv, _ := json.Marshal(ts)
					assoc_ts_map[tslot] = bv
				}
			}
		}

		//Atomically set all the values
		redisClient.Cmd("MULTI")
		redisClient.Cmd("SET", "ts:"+timeslot_id, booking_ts)
		for key, hash := range assoc_ts_map {
			redisClient.Cmd("SET", key, hash)
		}
		r := redisClient.Cmd("EXEC")

		if r.Err == nil {
			break
		}
	}

	redisClient.Close()
}

func main() {

	service.Init()

	service.Handle([]string{"/api/boats"}, BoatsHandler)
	service.Handle([]string{"/api/timeslots"}, TimeslotHandler)
	service.Handle([]string{"/api/assignments"}, AssignmentsHandler)
	service.Handle([]string{"/api/bookings"}, BookingHandler)

	//Start server
	fmt.Fprintf(os.Stderr, "error: %s", service.Start())
}
