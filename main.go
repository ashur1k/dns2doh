package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
)

func QueryType(qtype uint16) (string, bool) {
	types := map[uint16]string{
		0:     "None",
		1:     "A",
		2:     "NS",
		3:     "MD",
		4:     "MF",
		5:     "CNAME",
		6:     "SOA",
		7:     "MB",
		8:     "MG",
		9:     "MR",
		10:    "NULL",
		12:    "PTR",
		13:    "HINFO",
		14:    "MINFO",
		15:    "MX",
		16:    "TXT",
		17:    "RP",
		18:    "AFSDB",
		19:    "X25",
		20:    "ISDN",
		21:    "RT",
		23:    "NSAPPTR",
		24:    "SIG",
		25:    "KEY",
		26:    "PX",
		27:    "GPOS",
		28:    "AAAA",
		29:    "LOC",
		30:    "NXT",
		31:    "EID",
		32:    "NIMLOC",
		33:    "SRV",
		34:    "ATMA",
		35:    "NAPTR",
		36:    "KX",
		37:    "CERT",
		39:    "DNAME",
		41:    "OPT",
		42:    "APL",
		43:    "DS",
		44:    "SSHFP",
		45:    "IPSECKEY",
		46:    "RRSIG",
		47:    "NSEC",
		48:    "DNSKEY",
		49:    "DHCID",
		50:    "NSEC3",
		51:    "NSEC3PARAM",
		52:    "TLSA",
		53:    "SMIMEA",
		55:    "HIP",
		56:    "NINFO",
		57:    "RKEY",
		58:    "TALINK",
		59:    "CDS",
		60:    "CDNSKEY",
		61:    "OPENPGPKEY",
		62:    "CSYNC",
		63:    "ZONEMD",
		64:    "SVCB",
		65:    "HTTPS",
		99:    "SPF",
		100:   "UINFO",
		101:   "UID",
		102:   "GID",
		103:   "UNSPEC",
		104:   "NID",
		105:   "L32",
		106:   "L64",
		107:   "LP",
		108:   "EUI48",
		109:   "EUI64",
		249:   "TKEY",
		250:   "TSIG",
		251:   "IXFR",
		252:   "AXFR",
		253:   "MAILB",
		254:   "MAILA",
		255:   "ANY",
		256:   "URI",
		257:   "CAA",
		258:   "AVC",
		260:   "AMTRELAY",
		32768: "TA",
		32769: "DLV",
		65535: "Reserved",
	}

	t, ok := types[qtype]
	return t, ok
}

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

	// URL пользовательского DNS-over-HTTPS сервера
	serverURL := *doh

	for {
		buf := make([]byte, 512)
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Println(err)
			continue
		}

		// Извлекаем тип запроса DNS
		offset := 12
		for i, value := range buf[12:n] {
			log.Println(i, value)
			if value == byte(0) {
				offset = offset + i + 1
				break
			}
		}
		queryTypeBytes := binary.BigEndian.Uint16(buf[offset : offset+2])
		queryType, ok := QueryType(queryTypeBytes)

		// Задаем параметры запроса
		params := url.Values{}
		if ok {
			params.Add("queryType", queryType)
		}

		// Создаём запрос с заголовками X-Forwarded-For, X-Real-IP и другие
		var req *http.Request
		var errReq error
		if *useGet {
			// Кодируем запрос в base64
			dnsRequestBase64 := base64.RawURLEncoding.EncodeToString(buf[:n])
			dnsRequestBase64 = strings.TrimRight(dnsRequestBase64, "=")

			// Добавляем параметры запроса
			params.Add("dns", dnsRequestBase64)

			req, errReq = http.NewRequest("GET", serverURL, nil)
		} else {
			req, errReq = http.NewRequest("POST", serverURL, bytes.NewBuffer(buf[:n]))

		}

		if errReq != nil {
			fmt.Println("Error creating request:", err)
			continue
		}

		// Добавляем параметры запроса
		req.URL.RawQuery = params.Encode()

		// Добавляем заголовки
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
			// Если ответ не 403, то выводим ошибку
			if resp.StatusCode != http.StatusForbidden {
				log.Println("DNS-over-HTTPS request failed")
			}
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
