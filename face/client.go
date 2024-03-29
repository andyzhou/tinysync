package face

import (
	"context"
	"github.com/andyzhou/tinysync/define"
	pb "github.com/andyzhou/tinysync/pb"
	"google.golang.org/grpc"
	"log"
	"sync"
	"time"
)

/*
 * face for rpc client
 * @author <AndyZhou>
 * @mail <diudiu8848@163.com>
 */

//face info
type Client struct {
	addr string
	conn *grpc.ClientConn //rpc client connect
	client *pb.FileSyncServiceClient //rpc client
	dirSyncChan chan pb.DirSyncReq
	fileSyncChan chan pb.FileSyncReq
	fileRemoveChan chan pb.FileRemoveReq
	closeChan chan bool
	sync.RWMutex
}

//construct
func NewClient(addr string) *Client {
	//self init
	this := &Client{
		addr:addr,
		dirSyncChan:make(chan pb.DirSyncReq, define.ReqChanSize),
		fileSyncChan:make(chan pb.FileSyncReq, define.ReqChanSize),
		fileRemoveChan:make(chan pb.FileRemoveReq, define.ReqChanSize),
		closeChan:make(chan bool, 1),
	}

	//try connect server
	this.connServer()

	//spawn main process
	go this.runMainProcess()

	return this
}

//quit
func (f *Client) Quit() {
	f.closeChan <- true
}

//call api
func (f *Client) DirSync(
					subDir string,
					newSubDir string,
					isRemove bool,
				) (bRet bool) {
	//basic check
	if subDir == "" || f.client == nil {
		bRet = false
		return
	}

	//try catch panic
	defer func() {
		if err := recover(); err != nil {
			log.Println("Client::DirSync panic, err:", err)
			bRet = false
			return
		}
	}()

	//init request
	req := pb.DirSyncReq{
		SubDir:subDir,
		NewSubDir:newSubDir,
		IsRemove:isRemove,
	}

	//send to chan
	f.dirSyncChan <- req
	bRet = true
	return
}

func (f *Client) FileRemove(
					subDir string,
					fileName string,
				) (bRet bool) {
	//basic check
	if subDir == "" || fileName == "" || f.client == nil {
		bRet = false
		return
	}

	//try catch panic
	defer func() {
		if err := recover(); err != nil {
			log.Println("Client::DocRemove panic, err:", err)
			bRet = false
			return
		}
	}()

	//init request
	req := pb.FileRemoveReq{
		SubDir:subDir,
		FileName:fileName,
	}

	//send to chan
	f.fileRemoveChan <- req
	bRet = true
	return
}

func (f *Client) FileSync(
					req *pb.FileSyncReq,
				) (bRet bool) {
	//basic check
	if req == nil || f.client == nil {
		bRet = false
		return
	}

	//try catch panic
	defer func() {
		if err := recover(); err != nil {
			log.Println("Client::FileSync panic, err:", err)
			bRet = false
			return
		}
	}()

	//send to chan
	f.fileSyncChan <- *req
	bRet = true
	return
}


///////////////
//private func
///////////////

//run main process
func (f *Client) runMainProcess() {
	var (
		ticker = time.NewTicker(time.Second * define.ClientCheckTicker)
		syncReq pb.FileSyncReq
		removeReq pb.FileRemoveReq
		dirReq pb.DirSyncReq
		isOk, needQuit bool
	)

	//loop
	for {
		if needQuit {
			break
		}
		select {
		case syncReq, isOk = <- f.fileSyncChan://file sync req
			if isOk {
				f.fileSyncProcess(&syncReq)
			}
		case removeReq, isOk = <- f.fileRemoveChan://file remove req
			if isOk {
				f.fileRemoveProcess(&removeReq)
			}
		case dirReq, isOk = <- f.dirSyncChan://dir sync req
			if isOk {
				f.dirSyncProcess(&dirReq)
			}
		case <- ticker.C://check status
			{
				f.ping()
			}
		case <- f.closeChan:
			needQuit = true
		}
	}

	//close chan
	close(f.fileSyncChan)
	close(f.fileRemoveChan)
	close(f.dirSyncChan)
	close(f.closeChan)
}

//dir sync for rpc server
func (f *Client) dirSyncProcess(
					req *pb.DirSyncReq,
				) bool {
	if req == nil {
		return false
	}

	//call dir sync api
	resp, err := (*f.client).DirSync(
		context.Background(),
		req,
	)

	if err != nil {
		log.Println("Client::dirSyncProcess failed, err:", err.Error())
		return false
	}

	return resp.Success
}

//file remove from rpc server
func (f *Client) fileRemoveProcess(
					req *pb.FileRemoveReq,
				) bool {
	if req == nil {
		return false
	}

	//call file remove api
	resp, err := (*f.client).FileRemove(
		context.Background(),
		req,
	)

	if err != nil {
		log.Println("Client::fileRemoveProcess failed, err:", err.Error())
		return false
	}

	return resp.Success
}

//file sync into rpc server
func (f *Client) fileSyncProcess(
					req *pb.FileSyncReq,
				) bool {
	if req == nil {
		return false
	}

	//call file sync api
	resp, err := (*f.client).FileSync(
				context.Background(),
				req,
			)

	if err != nil {
		log.Println("Client::fileSyncProcess failed, err:", err.Error())
		return false
	}

	return resp.Success
}

//ping server
func (f *Client) ping() bool {
	//check status
	isOk := f.checkStatus()
	if isOk {
		return true
	}
	//try re connect
	f.connServer()
	return true
}

//check server status
func (f *Client) checkStatus() bool {
	//check connect
	if f.conn == nil {
		return false
	}
	//get status
	state := f.conn.GetState().String()
	if state == "TRANSIENT_FAILURE" || state == "SHUTDOWN" {
		return false
	}
	return true
}

//connect rpc server
func (f *Client) connServer() bool {
	//try connect
	conn, err := grpc.Dial(f.addr, grpc.WithInsecure())
	if err != nil {
		log.Println("Client::connServer failed, err:", err.Error())
		return false
	}

	//init rpc client
	client := pb.NewFileSyncServiceClient(conn)
	if client == nil {
		return false
	}

	//sync
	f.Lock()
	defer f.Unlock()
	f.conn = conn
	f.client = &client

	return true
}