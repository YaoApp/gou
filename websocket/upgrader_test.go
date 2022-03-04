package websocket

// var cstDialer = websocket.Dialer{
// 	Subprotocols:     []string{"p1", "p2"},
// 	ReadBufferSize:   1024,
// 	WriteBufferSize:  1024,
// 	HandshakeTimeout: 30 * time.Second,
// }

// func TestStart(t *testing.T) {
// 	log.SetOutput(os.Stdout)
// 	log.SetLevel(log.TraceLevel)
// 	ws := &WebSocket{}

// 	http.HandleFunc("/echo", func(rw http.ResponseWriter, r *http.Request) {
// 		ws.Start(rw, r, nil)
// 	})

// 	go func() { http.ListenAndServe("127.0.0.1:5081", nil) }()

// 	for i := 0; i < 3; i++ {
// 		time.Sleep(200 * time.Microsecond)
// 		conn, err := Dial()
// 		if err != nil || conn == nil {
// 			continue
// 		}
// 		echo(t, conn)
// 		break
// 	}
// }

// func Dial() (*websocket.Conn, error) {
// 	ws, _, err := cstDialer.Dial("ws://127.0.0.1:5081/echo", nil)
// 	if err != nil {
// 		log.Error("Dial: %v", err)
// 	}
// 	return ws, nil
// }

// func echo(t *testing.T, ws *websocket.Conn) {

// 	const message = "Hello World!"
// 	if err := ws.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
// 		t.Fatalf("SetWriteDeadline: %v", err)
// 	}
// 	if err := ws.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
// 		t.Fatalf("WriteMessage: %v", err)
// 	}
// 	if err := ws.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
// 		t.Fatalf("SetReadDeadline: %v", err)
// 	}
// 	_, p, err := ws.ReadMessage()
// 	if err != nil {
// 		t.Fatalf("ReadMessage: %v", err)
// 	}
// 	if string(p) != message {
// 		t.Fatalf("message=%s, want %s", p, message)
// 	}

// 	log.Trace("Message:%s", message)
// }
