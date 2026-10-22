package main

func okResp(data any) Response {
	return Response{Ok: true, Data: data}
}

func errResp(msg string) Response {
	return Response{Ok: false, Error: msg}
}
