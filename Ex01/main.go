//直接使用ListenAndServe使用fasthttp的例子
package main

import (
	"fmt"
	"github.com/valyala/fasthttp"

	"encoding/json"
	"strconv"
	"log"
)

type DataResult struct {
	State int32			`json:"state"`
	Message interface{}  `json:"msg"`
}

type TaskDataStore struct{
	Task string				`json:"task"`
	Market string			`json:"market"`
	Appid string			`json:"appid"`
}

type TaskData struct{
	Task string				`json:"task"`
	Market string			`json:"market"`
	FirstColor string		`json:"fc"`
	OffsetColor string		`json:"oc"`
}

type AppFeature struct {
	AppId string
	Orientation int
	AppFeatureString string
}

type SetTaskResponse struct{
	TaskStore TaskDataStore
	Features []AppFeature
}

type Element interface{}

type Queue interface {
	Offer(e Element)    //向队列中添加元素
	Poll()   Element    //移除队列中最前面的元素
	Clear()  bool       //清空队列
	Size()  int            //获取队列的元素个数
	IsEmpty() bool   //判断队列是否是空
}

type  sliceEntry struct{
	element []Element
}

func NewQueue() *sliceEntry {
	return &sliceEntry{}
}

//向队列中添加元素
func (entry *sliceEntry) Offer(e Element) {
	entry.element = append(entry.element,e)
}

//移除队列中最前面的额元素
func (entry *sliceEntry) Poll() Element {
	if entry.IsEmpty() {
		fmt.Println("queue is empty!")
		return nil
	}

	firstElement := entry.element[0]
	entry.element = entry.element[1:]
	return firstElement
}

func (entry *sliceEntry) Clear() bool {
	if entry.IsEmpty() {
		fmt.Println("queue is empty!")
		return false
	}
	for i:=0 ; i< entry.Size() ; i++ {
		entry.element[i] = nil
	}
	entry.element = nil
	return true
}

func (entry *sliceEntry) Size() int {
	return len(entry.element)
}

func (entry *sliceEntry) IsEmpty() bool {
	if len(entry.element) == 0 {
		return true
	}
	return false
}


var taskQueue = NewQueue()


const (
	ERROR_SUCCESS			= 0
	ERROR_PARAMS_INVALID	= -30001
	ERROR_SQL 				= -30002
	ERROR_NOTIMPL			= -30003
	ERROR_TASK_APPID		= -30004
)

func jsonResult(ctx *fasthttp.RequestCtx, state int32, msg interface{}) {
	json.NewEncoder(ctx).Encode(DataResult{state,msg})
}

func JsonMsgResult(ctx *fasthttp.RequestCtx, msg interface{} ) {
	jsonResult(ctx, ERROR_SUCCESS, msg)
}

func JsonSuccResult(ctx *fasthttp.RequestCtx) {
	jsonResult(ctx, ERROR_SUCCESS, "success")
}

func JsonErrorResult(ctx *fasthttp.RequestCtx, state int32, msg string) {
	jsonResult(ctx, state, msg)
}



// RequestHandler 类型，使用 RequestCtx 传递 HTTP 的数据
func httpHandle(ctx *fasthttp.RequestCtx) {
	path := string(ctx.Path())
	if  path == "/get"	{
		deviceid := string(ctx.FormValue("deviceid"))
		devicemodel := string(ctx.FormValue("devicemodel"))
		orientation := string(ctx.FormValue("orientation"))
		orient, err := strconv.Atoi(orientation)
		if err != nil {
			orient = 0
			fmt.Fprintf(ctx, "屏幕方向参数错误%s",orientation)
			return
		}

		fmt.Printf("deviceid=%s, devicemodel=%s\n",deviceid,devicemodel)
		fmt.Println("Has task:",!taskQueue.IsEmpty())

		if !taskQueue.IsEmpty() {
			taskStore := taskQueue.Poll().(TaskDataStore)
			fc, oc := queryFeature(taskStore.Appid,orient)

			task := TaskData{taskStore.Task, taskStore.Market, fc, oc }
			json.NewEncoder(ctx).Encode(task)
			fmt.Println("Reponse:",task)
		}
	}else if path == "/set" {
		t := string(ctx.FormValue("task"))
		m := string(ctx.FormValue("market"))
		appid := string(ctx.FormValue("id"))

		if len(m) <= 0 {
			m = "QQDownloader"
		}
		if len(t) > 0 {
			taskStore := TaskDataStore{t,m,appid}

			features, err := queryFeatures(appid)
			if err != nil {
				JsonErrorResult(ctx,ERROR_SQL,fmt.Sprintf( "查询%s特征出现错误,不添加任务", appid))
				return
			}

			if len(features) > 0 {
				taskQueue.Offer(taskStore)
				JsonMsgResult(ctx,SetTaskResponse{taskStore, features})
			}else {
				JsonErrorResult(ctx,ERROR_SQL,fmt.Sprintf("数据库没有%s对应的特征字符串",appid))
				return
			}

			fmt.Println("Reponse:",SetTaskResponse{taskStore, features})
		}
	}else if path == "/" {
		ctx.SendFile("home.html")
	}
}

func main() {

	if err := OpenDatabase("ebkadmin:ebkadmin@tcp(192.168.1.77:3306)/ebk?charset=utf8" ); err != nil {
		log.Println( "Error OpenDatabase:", err )
		return
	}
	defer CloseDatabase()

	// 一定要写 httpHandle，否则会有 nil pointer 的错误，没有处理 HTTP 数据的函数
	if err := fasthttp.ListenAndServe("0.0.0.0:12345", httpHandle); err != nil {
		fmt.Println("start fasthttp fail:", err.Error())
	}
}


