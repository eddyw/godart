// The program rpcuse drives net/rpc and net/rpc/jsonrpc. Each listener binds
// port 0, and the program reads the real address, thus the output holds no
// port of the machine.
package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"sort"
	"strings"
)

// Args holds the 2 operands of a call.
type Args struct {
	A, B int
}

// Quotient holds the 2 results of a division.
type Quotient struct {
	Quo, Rem int
}

// Arith holds each method of the service.
type Arith int

// Multiply writes the product.
func (t *Arith) Multiply(args *Args, reply *int) error {
	*reply = args.A * args.B
	return nil
}

// Divide writes the quotient and the rest.
func (t *Arith) Divide(args *Args, quo *Quotient) error {
	if args.B == 0 {
		return errors.New("divide by zero")
	}
	quo.Quo = args.A / args.B
	quo.Rem = args.A % args.B
	return nil
}

// Echo gives the text again in upper case.
type Echo struct{}

// Upper writes the text in upper case.
func (e *Echo) Upper(in *string, out *string) error {
	*out = strings.ToUpper(*in)
	return nil
}

// Empty holds no method of a suitable shape.
type Empty struct{}

func main() {
	// The log of the package writes the cause of a bad Register. The flags of
	// the date leave the output, thus the 2 builds write the same text.
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	gobPath()
	jsonPath()
	httpPath()
	values()
}

// gobPath drives the gob codec of the package over TCP.
func gobPath() {
	fmt.Println("== gob ==")
	server := rpc.NewServer()
	fmt.Println("register", server.Register(new(Arith)))
	fmt.Println("register again", server.Register(new(Arith)))
	fmt.Println("register name", server.RegisterName("Text", new(Echo)))
	fmt.Println("register empty", server.Register(new(Empty)))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("listen:", err)
		return
	}
	addr := ln.Addr().String()
	go accept(ln, server)

	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		fmt.Println("dial:", err)
		return
	}
	var product int
	fmt.Println("multiply", client.Call("Arith.Multiply", &Args{6, 7}, &product), product)

	var quo Quotient
	fmt.Println("divide", client.Call("Arith.Divide", &Args{17, 5}, &quo), quo)
	fmt.Println("divide 0", client.Call("Arith.Divide", &Args{17, 0}, &quo))

	var upper string
	in := "hello"
	fmt.Println("upper", client.Call("Text.Upper", &in, &upper), upper)

	fmt.Println("no method", client.Call("Arith.Nope", &Args{1, 2}, &product))
	fmt.Println("no service", client.Call("Nope.Nope", &Args{1, 2}, &product))
	fmt.Println("no dot", client.Call("Nope", &Args{1, 2}, &product))

	// The asynchronous form. The 3 calls come back in the order of the queue.
	done := make(chan *rpc.Call, 3)
	var a, b, c int
	client.Go("Arith.Multiply", &Args{2, 3}, &a, done)
	client.Go("Arith.Multiply", &Args{4, 5}, &b, done)
	client.Go("Arith.Multiply", &Args{6, 7}, &c, done)
	var got []string
	for i := 0; i < 3; i++ {
		call := <-done
		reply := call.Reply.(*int)
		got = append(got, fmt.Sprintf("%s=%d err=%v", call.ServiceMethod, *reply, call.Error))
	}
	sort.Strings(got)
	for _, g := range got {
		fmt.Println("async", g)
	}

	fmt.Println("close", client.Close())
	fmt.Println("close again", client.Close())
	fmt.Println("after close", client.Call("Arith.Multiply", &Args{1, 1}, &product))
	fmt.Println("shutdown is ErrShutdown",
		errors.Is(client.Call("Arith.Multiply", &Args{1, 1}, &product), rpc.ErrShutdown))
	ln.Close()
}

// jsonPath drives the JSON codec of net/rpc/jsonrpc.
func jsonPath() {
	fmt.Println("== jsonrpc ==")
	server := rpc.NewServer()
	if err := server.Register(new(Arith)); err != nil {
		fmt.Println("register:", err)
		return
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("listen:", err)
		return
	}
	addr := ln.Addr().String()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go server.ServeCodec(jsonrpc.NewServerCodec(conn))
		}
	}()

	client, err := jsonrpc.Dial("tcp", addr)
	if err != nil {
		fmt.Println("dial:", err)
		return
	}
	var product int
	fmt.Println("multiply", client.Call("Arith.Multiply", &Args{3, 4}, &product), product)
	var quo Quotient
	fmt.Println("divide", client.Call("Arith.Divide", &Args{9, 4}, &quo), quo)
	fmt.Println("divide 0", client.Call("Arith.Divide", &Args{9, 0}, &quo))
	var serverErr rpc.ServerError
	err = client.Call("Arith.Nope", &Args{1, 1}, &product)
	fmt.Println("no method is a ServerError", errors.As(err, &serverErr), err)
	fmt.Println("close", client.Close())
	ln.Close()
}

// httpPath drives the CONNECT form of the package over HTTP.
func httpPath() {
	fmt.Println("== http ==")
	server := rpc.NewServer()
	if err := server.Register(new(Arith)); err != nil {
		fmt.Println("register:", err)
		return
	}
	mux := http.NewServeMux()
	mux.Handle(rpc.DefaultRPCPath, server)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("listen:", err)
		return
	}
	addr := ln.Addr().String()
	go http.Serve(ln, mux)

	client, err := rpc.DialHTTP("tcp", addr)
	if err != nil {
		fmt.Println("dial http:", err)
		return
	}
	var product int
	fmt.Println("multiply", client.Call("Arith.Multiply", &Args{8, 9}, &product), product)
	fmt.Println("close", client.Close())

	// A path that holds no RPC server gives an OpError of net.
	_, err = rpc.DialHTTPPath("tcp", addr, "/nope")
	var op *net.OpError
	fmt.Println("bad path is an OpError", errors.As(err, &op))
	if op != nil {
		// op.Net holds the address, and the port of a listener of port 0
		// changes at each run. The program writes the network alone.
		network, _, _ := strings.Cut(op.Net, " ")
		fmt.Printf("op=%q net=%q err=%v\n", op.Op, network, op.Err)
	}
	ln.Close()
}

// accept holds the accept loop of the gob server.
func accept(ln net.Listener, server *rpc.Server) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go server.ServeConn(conn)
	}
}

// values reads each name of the package that needs no server.
func values() {
	fmt.Println("== values ==")
	fmt.Println("rpc path", rpc.DefaultRPCPath)
	fmt.Println("debug path", rpc.DefaultDebugPath)
	fmt.Println("shutdown", rpc.ErrShutdown)
	var e rpc.ServerError = "the cause"
	fmt.Println("server error", e.Error())
	req := rpc.Request{ServiceMethod: "Arith.Multiply", Seq: 7}
	fmt.Printf("request %s %d\n", req.ServiceMethod, req.Seq)
	resp := rpc.Response{ServiceMethod: "Arith.Multiply", Seq: 7, Error: "none"}
	fmt.Printf("response %s %d %s\n", resp.ServiceMethod, resp.Seq, resp.Error)
	call := rpc.Call{ServiceMethod: "Arith.Multiply"}
	fmt.Println("call", call.ServiceMethod)
	fmt.Println("default server", rpc.DefaultServer != nil)
}
