package main

import (
	"fmt"
	"gopkg.in/redis.v5"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
        "database/sql"
        _ "github.com/lib/pq"
)


type Stat struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}


func main() {
        appspacelist,err:=getAppSpaceList("akkeris-redis")
        if err != nil {
            fmt.Println(err)
            os.Exit(2)
        }
        fmt.Println(appspacelist)

	for key, element := range appspacelist {
                getRedisStats(key, element)
	}


        appspacelist,err=getAppSpaceList("akkeris-memcached")
        if err != nil {
            fmt.Println(err)
            os.Exit(2)
        }
        fmt.Println(appspacelist)

        for key, element := range appspacelist {
                getMemcachedStats(key, element)
        }

}

func getRedisStats(bind string, app string) {
        brokerdb := os.Getenv("BROKERDB")
        uri := brokerdb
        db, dberr := sql.Open("postgres", uri)
        if dberr != nil {
                fmt.Println(dberr)
        }
        var name string
        var endpoint string
        dberr = db.QueryRow("select name, endpoint from resources where id = '"+bind+"'").Scan(&name, &endpoint)

        if dberr != nil {
                db.Close()
                fmt.Println(dberr)
        }
        db.Close()
        endpoint = strings.Replace(endpoint, "redis://","",-1)
        fmt.Println(endpoint)
		fmt.Println(name)
		var stats []Stat
		tsdbconn, err := net.Dial("tcp", os.Getenv("OPENTSDB_IP"))
		if err != nil {
			fmt.Println("dial error:", err)
			return
		}
		client := redis.NewClient(&redis.Options{
			Addr: endpoint,
		})
	
		result := client.Info()
		resultstring := fmt.Sprintf("%v", result)
		var resulta []string
		resulta = strings.Split(string(resultstring), "\n")
		for _, element := range resulta {
			if strings.Contains(element, ":") {
				var stat Stat
				stata := strings.Split(element, ":")
				stat.Key = stata[0]
				t := strings.TrimSpace(stata[1])
				stat.Value = t
				if stata[0] == "db0" {
					stat.Key = "keys"
					stat.Value = strings.Split(strings.Split(t, ",")[0], "=")[1]
				}
				if stat.Key == "connected_clients" || stat.Key == "used_memory" || stat.Key == "keys" {
					stats = append(stats, stat)
				}
			}
		}
		for _, element := range stats {
			key := element.Key
			metricname := "redis.info." + key
			value := element.Value
			timestamp := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
			host := name
			put := "put " + metricname + " " + timestamp + " " + value + " instance=" + host
					if app != "" {
						 put = put + " app="+app
					} 
					put = put +"\n"
					fmt.Println(put)
			fmt.Fprintf(tsdbconn, put)
		}
			tsdbconn.Close()
}


func getMemcachedStats(bind string, app string) {
	brokerdb := os.Getenv("BROKERDB")
	uri := brokerdb
	db, dberr := sql.Open("postgres", uri)
	if dberr != nil {
			fmt.Println(dberr)
	}
	var name string
	var endpoint string
	dberr = db.QueryRow("select name, endpoint from resources where id = '"+bind+"'").Scan(&name, &endpoint)

	if dberr != nil {
			db.Close()
			fmt.Println(dberr)
	}
	db.Close()
	fmt.Println(endpoint)
	fmt.Println(name)
	var stats []Stat
	tsdbconn, err := net.Dial("tcp", os.Getenv("OPENTSDB_IP"))
	if err != nil {
		fmt.Println("dial error:", err)
		return
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp4", endpoint)
	if err != nil {
		fmt.Println(err)
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		fmt.Println(err)
	}
	_, err = conn.Write([]byte("stats\n"))
	conn.CloseWrite()
	if err != nil {
		fmt.Println(err)
	}
	var resulta []string
	result, err := ioutil.ReadAll(conn)
	if err != nil {
		fmt.Println(err)
	}
	resulta = strings.Split(string(result), "\n")
	for _, element := range resulta {
		if strings.HasPrefix(element, "STAT") {
			var stat Stat
			stata := strings.Split(element, " ")
			if stata[1] == "bytes" || stata[1] == "curr_items" || stata[1] == "curr_connections" {
				stat.Key = stata[1]
				t := strings.TrimSpace(stata[2])
				stat.Value = t
				stats = append(stats, stat)
			}
		}
	}
	for _, element := range stats {
		key := element.Key
		metricname := "memcached.stat." + key
		value := element.Value
		timestamp := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
		host := name
		put := "put " + metricname + " " + timestamp + " " + value + " instance=" + host
                if app != "" {
                    put = put + " app="+app
                }
                put = put +"\n"
                fmt.Println(put)
		fmt.Fprintf(tsdbconn, put)
	}
        tsdbconn.Close()
 
}


func getAppSpaceList(engine string) (l map[string]string, e error) {
        var list map[string]string
        list = make(map[string]string)


        brokerdb := os.Getenv("PITDB")
        uri := brokerdb
        db, dberr := sql.Open("postgres", uri)
        if dberr != nil {
                fmt.Println(dberr)
                return nil, dberr
        }
        stmt,dberr := db.Prepare("select apps.name as appname, spaces.name as space, service_attachments.service as bindname from apps, spaces, services, service_attachments where services.service=service_attachments.service and owned=true and addon_name='"+engine+"' and services.deleted=false and service_attachments.deleted=false and service_attachments.app=apps.app and spaces.space=apps.space;")
        defer stmt.Close()
        rows, err := stmt.Query()
        if dberr != nil {
                db.Close()
                fmt.Println(dberr)
                return nil, dberr
        }
        defer rows.Close()
        var appname string
        var space string
        var bindname string
        for rows.Next() {
                err := rows.Scan(&appname, &space, &bindname)
                if err != nil {
                        fmt.Println(err)
                        return nil, err
                }
                list[bindname]=appname+"-"+space
        }
        err = rows.Err()
        if err != nil {
                fmt.Println(err)
                return nil, err
        }
        db.Close()
        return list, nil
}

