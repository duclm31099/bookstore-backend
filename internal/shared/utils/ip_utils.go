package utils

import (
	"net"
	"strings"

	"github.com/gin-gonic/gin"
)

// ExtractClientIP extracts the real client IP address from the request.
// It handles various proxy scenarios and header combinations.
//
// Priority order:
// 1. X-Forwarded-For header (standard proxy header, takes first IP)
// 2. X-Real-IP header (nginx/cloudflare)
// 3. Direct connection RemoteAddr (fallback)
//
// Returns: Valid IP address string
func ExtractClientIP(c *gin.Context) string {
	// Try X-Forwarded-For first (proxy/load balancer)
	// Format: "client, proxy1, proxy2"
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		// Take the first IP (actual client)
		ips := strings.Split(xff, ",")
		clientIP := strings.TrimSpace(ips[0])

		// Validate IP format
		if isValidIP(clientIP) {
			return clientIP
		}
	}

	// Try X-Real-IP (nginx, cloudflare)
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		if isValidIP(xri) {
			return xri
		}
	}

	// Fallback to direct connection
	// RemoteAddr format: "IP:port" or "[IPv6]:port"
	remoteAddr := c.Request.RemoteAddr
	ip, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		// If no port, assume it's just an IP
		ip = remoteAddr
	}

	// Validate and return
	if isValidIP(ip) {
		return ip
	}

	// Ultimate fallback (should rarely happen)
	return "127.0.0.1"
}

// isValidIP validates if a string is a valid IPv4 or IPv6 address
func isValidIP(ip string) bool {
	if ip == "" {
		return false
	}

	// Parse and validate
	parsed := net.ParseIP(ip)
	return parsed != nil
}

// IsPrivateIP checks if an IP address is in private range
// Useful for detecting local/internal requests
func IsPrivateIP(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}

	// Check for IPv4 private ranges
	privateIPBlocks := []string{
		"10.0.0.0/8",     // Private network
		"172.16.0.0/12",  // Private network
		"192.168.0.0/16", // Private network
		"127.0.0.0/8",    // Loopback
	}

	for _, cidr := range privateIPBlocks {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if block.Contains(parsed) {
			return true
		}
	}

	// Check for IPv6 loopback
	if parsed.IsLoopback() {
		return true
	}

	return false
}

// GetIPInfo returns detailed information about the IP address
// Used for logging and debugging
func GetIPInfo(c *gin.Context) map[string]interface{} {
	clientIP := ExtractClientIP(c)

	return map[string]interface{}{
		"client_ip":       clientIP,
		"is_private":      IsPrivateIP(clientIP),
		"x_forwarded_for": c.GetHeader("X-Forwarded-For"),
		"x_real_ip":       c.GetHeader("X-Real-IP"),
		"remote_addr":     c.Request.RemoteAddr,
	}
}
