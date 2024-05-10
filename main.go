package main

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
)

func main() {

	useGet := flag.Bool("get", false, "Set to use a GET request.")
	ip6 := flag.Bool("ip6", false, "Set to use a with IPv6 request.")
	doh := flag.String("doh", "https://dns.google/dns-query", "Set a custom DoH server")
	port := flag.Int("port", 53, "Set the DNS port")
	address := flag.String("address", "0.0.0.0", "Set the IP address")

	flag.Parse()

	// Выбираем, следует ли использовать IPv6
	addressType := "udp4"
	if *ip6 {
		addressType = "udp"
	}

	// Запускаем сервер на порту 53 для прослушивания DNS запросов
	conn, err := net.ListenUDP(addressType, &net.UDPAddr{IP: net.ParseIP(*address), Port: *port})
	if err != nil {
		log.Fatal(err)
	}
	// Получаем локальный адрес
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	log.Println("Listening on", localAddr.IP.String())

	defer conn.Close()

	// Создаём объект клиента
	client := &http.Client{}
	defer client.CloseIdleConnections()

	for {
		buf := make([]byte, 512)
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Println(err)
			continue
		}

		// URL пользовательского DNS-over-HTTPS сервера
		serverURL := *doh

		// Создаём запрос с заголовками X-Forwarded-For, X-Real-IP и другие
		var req *http.Request
		var errReq error
		if *useGet {
			// Кодируем запрос в base64
			dnsRequestBase64 := base64.RawURLEncoding.EncodeToString(buf[:n])
			dnsRequestBase64 = strings.TrimRight(dnsRequestBase64, "=")

			// Задаем параметры запроса
			params := url.Values{}
			params.Add("dns", dnsRequestBase64)

			// Формируем URL запроса для GET запроса
			requestURL := fmt.Sprintf("%s?%s", serverURL, params.Encode())

			req, errReq = http.NewRequest("GET", requestURL, nil)
		} else {
			req, errReq = http.NewRequest("POST", serverURL, bytes.NewBuffer(buf[:n]))

		}
		if errReq != nil {
			fmt.Println("Error creating request:", err)
			return
		}
		req.Header.Set("X-Forwarded-For", addr.IP.String())
		req.Header.Set("X-Real-IP", addr.IP.String())
		req.Header.Set("Host", localAddr.String())
		req.Header.Set("Accept", "application/dns-message")
		req.Header.Set("Content-Type", "application/dns-message")
		req.Header.Set("scheme", "application/dns-message")

		// Отправляем запрос в DNS-over-HTTPS
		resp, err := client.Do(req)
		if err != nil {
			log.Println(err)
			continue
		}

		// Если ответ не успешный, возвращаем ошибку
		if resp.StatusCode != http.StatusOK {
			log.Println("DNS-over-HTTPS request failed")
			resp.Body.Close()
			continue
		}

		// Читаем ответ
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		// Отправляем ответ обратно клиенту, который сделал DNS запрос
		conn.WriteToUDP(bodyBytes, addr)
	}
}
