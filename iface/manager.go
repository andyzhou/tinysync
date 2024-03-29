package iface

import pb "github.com/andyzhou/tinysync/pb"

/*
 * interface for inter manager
 * @author <AndyZhou>
 * @mail <diudiu8848@163.com>
 */

type IManager interface {
	Quit()

	//for dir sync
	DirSync(subDir, newSubDir string, isRemove bool, cb func(subDir, newSubDir string, isRemove bool)) bool

	//for file sync and remove
	FileRemove(subDir, fileName string, cb func(subDir, fileName string)) bool
	FileSync(req *pb.FileSyncReq, cb func(subDir, fileName string)) bool

	//for rpc node
	RemoveNode(addr string) bool
	AddNode(addr string) bool
}