# Caddy GoDaddy DNS Provider Plugin

A Caddy v2 DNS provider plugin that enables ACME DNS-01 challenge validation through the GoDaddy DNS API to obtain TLS certificates from Let's Encrypt or other ACME CAs.

## Features

- DNS-01 ACME challenge validation support
- Wildcard domain certificate support
- Automatic DNS TXT record creation and deletion
- Configurable API timeout
- Environment variable configuration support

## Installation

### Method 1: Build with xcaddy

First, install xcaddy:
```bash
go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest
```

Then build Caddy with the GoDaddy DNS plugin:
```bash
xcaddy build --with github.com/xcaddyplugins/caddy-dns-godaddy
```

### Method 2: Docker Build

Create a Dockerfile:
```dockerfile
FROM caddy:builder AS builder
RUN xcaddy build \
    --with github.com/xcaddyplugins/caddy-dns-godaddy

FROM caddy:latest
COPY --from=builder /usr/bin/caddy /usr/bin/caddy
```

## Configuration

### Obtaining GoDaddy API Credentials

1. Log in to the GoDaddy Developer Portal: https://developer.godaddy.com/
2. Create a new API key
3. Record the API Key and API Secret

### Caddyfile Configuration

```caddyfile
{
	email your-email@example.com
}

example.com {
	tls {
		dns godaddy {
			api_key {env.GODADDY_API_KEY}
			api_secret {env.GODADDY_API_SECRET}
			http_timeout 30s  # Optional, defaults to 30 seconds
		}
	}

	respond "Hello World!"
}
```

### Environment Variables

```bash
export GODADDY_API_KEY="your_api_key_here"
export GODADDY_API_SECRET="your_api_secret_here"
```

### JSON Configuration

If you're using JSON format for Caddy configuration:
```json
{
  "apps": {
    "http": {
      "servers": {
        "srv0": {
          "listen": [":443"],
          "routes": [
            {
              "match": [{"host": ["example.com"]}],
              "handle": [
                {
                  "handler": "static_response",
                  "body": "Hello World!"
                }
              ],
              "terminal": true
            }
          ]
        }
      }
    },
    "tls": {
      "automation": {
        "policies": [
          {
            "subjects": ["example.com"],
            "issuers": [
              {
                "module": "acme",
                "challenges": {
                  "dns": {
                    "provider": {
                      "name": "godaddy",
                      "api_key": "{env.GODADDY_API_KEY}",
                      "api_secret": "{env.GODADDY_API_SECRET}"
                    }
                  }
                }
              }
            ]
          }
        ]
      }
    }
  }
}
```

## Configuration Options

| Option | Required | Default | Description |
|--------|----------|---------|-------------|
| `api_key` | Yes | None | GoDaddy API Key |
| `api_secret` | Yes | None | GoDaddy API Secret |
| `http_timeout` | No | 30s | HTTP request timeout |

## Usage Examples

### Single Domain Certificate
```caddyfile
example.com {
	tls {
		dns godaddy {
			api_key {env.GODADDY_API_KEY}
			api_secret {env.GODADDY_API_SECRET}
		}
	}
	respond "Single domain"
}
```

### Wildcard Certificate
```caddyfile
*.example.com {
	tls {
		dns godaddy {
  		api_key {env.GODADDY_API_KEY}
  		api_secret {env.GODADDY_API_SECRET}
  	}
  }
  respond "Wildcard domain"
}
```

### Multi-Domain Certificate
```caddyfile
example.com, www.example.com, api.example.com {
	tls {
		dns godaddy {
			api_key {env.GODADDY_API_KEY}
			api_secret {env.GODADDY_API_SECRET}
		}
	}
	respond "Multiple domains"
}
```

## Troubleshooting

### Common Errors

1. **API Authentication Failed**
   - Verify that your API key and secret are correct
   - Ensure the API key has necessary DNS management permissions

2. **Domain Mismatch**
   - Confirm the domain is managed in your GoDaddy account
   - Check domain spelling for accuracy

3. **Network Timeout**
   - Increase the `http_timeout` value
   - Check network connectivity

### Debug Mode

Enable Caddy debug logging:
```bash
caddy run --config Caddyfile --adapter caddyfile --debug
```

### DNS Propagation Issues

If DNS challenges are failing:
- Wait for DNS propagation (can take up to 10 minutes)
- Verify TXT records are being created correctly
- Check GoDaddy DNS management interface

## API Rate Limits

GoDaddy imposes rate limits on their API:
- Production: 60 requests per minute
- OTE (testing): 60 requests per minute

The plugin automatically handles rate limiting with appropriate delays.

## Security Considerations

- Store API credentials securely using environment variables
- Use production API keys only in production environments
- Regularly rotate API credentials
- Monitor API usage in GoDaddy developer dashboard

## Testing

### Unit Tests
```bash
go test ./...
```

### Integration Testing
Set up test environment variables:
```bash
export GODADDY_API_KEY="test_key"
export GODADDY_API_SECRET="test_secret"
export TEST_DOMAIN="yourdomain.com"
go test -tags=integration ./...
```

## Requirements

- Go 1.21 or later
- Valid GoDaddy domain and API credentials
- Internet connectivity for ACME and DNS API access

## Compatibility

- Caddy v2.7.0 and later
- libdns v0.2.1 and later
- Go modules support

## Performance

The plugin is optimized for:
- Minimal API calls during certificate issuance
- Efficient DNS record cleanup
- Concurrent certificate requests handling
- Low memory footprint

## Monitoring

Enable structured logging to monitor plugin activity:
```json
{
  "logging": {
    "logs": {
      "default": {
        "level": "INFO",
        "writer": {
          "output": "stdout"
        }
      }
    }
  }
}
```

## Contributing

Contributions are welcome! Please ensure:
1. Code follows Go coding standards
2. Include appropriate tests
3. Update documentation as needed
4. Follow semantic versioning for releases

### Development Setup

1. Fork the repository
2. Clone your fork
3. Install dependencies: `go mod download`
4. Make changes
5. Run tests: `go test ./...`
6. Submit a pull request

## License

This project is licensed under the MIT License. See the LICENSE file for details.

## Support

For support:
1. Check this README documentation
2. Search existing GitHub Issues
3. Create a new Issue with detailed information
4. Join the Caddy Community Forum for general Caddy questions

## Changelog

### v1.0.0
- Initial release
- Basic GoDaddy DNS provider functionality
- ACME DNS-01 challenge support
- Wildcard certificate support

## Related Projects

- [Caddy](https://caddyserver.com/) - The web server this plugin extends
- [libdns](https://github.com/libdns/libdns) - DNS provider interface
- [GoDaddy API Documentation](https://developer.godaddy.com/doc/endpoint/domains)

## Acknowledgments

- Caddy development team for the excellent plugin architecture
- libdns project for standardized DNS provider interfaces
- GoDaddy for providing a comprehensive DNS API