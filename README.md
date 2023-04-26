# DNS-over-HTTPS proxy server in Golang

This project is a proxy server that allows to redirect DNS queries through a DoH (DNS-over-HTTPS) server. The server listens for requests on port 53 and sends them to the specified DoH server to get a response. Then the response is sent back to the original request address.

## Dependencies

To run the server, Go version 1.15 or higher is required.

## Installation

1. Clone the repository:

   ```sh
   git clone https://github.com/ashur1k/dns2doh.git
   ```

2. Navigate to the project directory:

   ```sh
   cd dns2doh
   ```

3. Build the project:

   ```sh
   go build
   ```

## Usage

### Command-line flags

```text
-get
```

Set to use GET requests instead of POST for sending DNS queries.

```text
-ip6
```

Set to use IPv6 requests instead of IPv4.

```text
-doh
```

Set a custom DoH server to handle requests. The default is `https://dns.google/dns-query`.

```text
-port
```

Set the port to listen for DNS queries. The default is port 53.

```text
-address
```

Set the IP address to listen for DNS queries. The default is `0.0.0.0`.

### Starting the server

1. Start the server:

   ```sh
   ./dns2doh
   ```

2. Send a DNS query to the server's port 53.

## Contribution

If you have suggestions or comments on the code, please create an issue or pull request.