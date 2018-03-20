package main

import (
	"fmt"
	"os"
	"flag"
	"net"
	"io"
	"strconv"
	"encoding/binary"
)

func main() {
	var host = flag.String("host","","host")
	var port = flag.String("port","8889","port")

	flag.Parse()

	addr,err := net.ResolveTCPAddr("tcp",*host+":"+*port)
	if err != nil{
		fmt.Println("Can't resolve address:",err)
		os.Exit(1)
	}
	l,err := net.ListenTCP("tcp",addr)
	if err != nil{
		fmt.Println("Error listening:",err)
		os.Exit(1)
	}
	defer l.Close()
	fmt.Println("Listening on "+*host+":"+*port)


	for{
		conn,err:=l.AcceptTCP()
		if err != nil{
			fmt.Println("Error accepting:",err)
			continue
		}
		fmt.Printf("Accepted connection %s -> %s \n",conn.RemoteAddr(),conn.LocalAddr())
		conn.SetLinger(0)
		go handleConn(conn)
	}

}


func handleConn(conn *net.TCPConn){
	defer conn.Close()

	buf := make([]byte,256)

	//first step
	n,err := conn.Read(buf)
	if err != nil{
		fmt.Println(err)
		return
	}
	if buf[0] != 0x05 {
		fmt.Println("only support socks5")
		return
	}
	conn.Write([]byte{0x05,0x00})

	//second step
	n,err = conn.Read(buf)
	if err != nil{
		fmt.Println(err)
		return
	}
	if n < 7 {
		//todo : use io.ReadFull
		fmt.Println("half pack occurs")
		return
	}
	if buf[1] != 0x01 {
		fmt.Println("only support connect")
		return
	}
	var dIp []byte
	switch buf[3]{
	case 0x01:
		//ip v4 address
		dIp = buf[4:4+net.IPv4len]
	case 0x03:
		//domain name return
		dIp = buf[5:n-2]
	case 0x04:
		//ip v6 address
		dIp = buf[4:4+net.IPv6len]
	default:
		fmt.Println("unknown remote server address type")
		return
	}
	dPort := strconv.Itoa(int(binary.BigEndian.Uint16(buf[n-2:n])))

	dstAddr := net.IP(dIp).String()+":"+ dPort
	fmt.Println("dial "+dstAddr)
	dstServer,err := net.Dial("tcp",dstAddr)
	if err != nil{
		fmt.Println("connect remote server error",err)
		return
	}
	defer dstServer.Close()
	conn.Write([]byte{0x05,0x00,0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	//third step
	go func(){
		_,err = io.Copy(dstServer,conn)
		if err != nil{
			fmt.Println(err)
			conn.Close()
			dstServer.Close()
		}
	}()
	io.Copy(conn,dstServer)
}