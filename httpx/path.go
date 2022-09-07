package httpx

import (
	"path/filepath"
)

func lastChar(s string) uint8 {
	if s == "" {
		return 0
	}
	return s[len(s)-1]
}

// 拼接绝对路径和相对路径
func joinPath(absolutePath, relativePath string) string {
	if relativePath == "" {
		return absolutePath
	}

	finalPath := filepath.Join(absolutePath, relativePath)
	if lastChar(relativePath) == '/' && lastChar(finalPath) != '/' {
		finalPath += "/"
	}
	return finalPath
}

func cleanPath(path string) string {
	//默认切片容量
	var bufferSize = 128
	if path == "" {
		return "/"
	}

	buf := make([]byte, 0, bufferSize)
	n := len(path)

	r, w := 1, 1
	if path[0] != '/' {
		r = 0
		if n+1 > bufferSize {
			buf = make([]byte, 0, n+1)
		} else {
			buf = buf[:n+1]
		}

		buf[0] = '/'
	}

	trailing := n > 1 && path[n-1] == '/'

	for r < n {
		switch {
		case path[r] == '/':
			// /
			r++
		case path[r] == '.' && r+1 == n:
			trailing = true
			r++
		case path[r] == '.' && path[r+1] == '/':
			// ./
			r += 2
		case path[r] == '.' && path[r+1] == '.' && (r+2 == n || path[r+2] == '/'):
			// ..element or ../
			r += 3

			if w > 1 {
				// can backtrack
				w--
				if len(buf) == 0 {
					for w > 1 && path[w] != '/' {
						w--
					}
				} else {
					for w > 1 && buf[w] != '/' {
						w--
					}
				}
			}
		default:
			if w > 1 {
				lazyBuffer(&buf, path, w, '/')
				w++
			}

			for r < n && path[r] != '/' {
				lazyBuffer(&buf, path, w, path[r])
				w++
				r++
			}
		}
	}

	if trailing && w > 1 {
		lazyBuffer(&buf, path, w, '/')
		w++
	}

	if len(buf) == 0 {
		return path[:w]
	}
	return string(buf[:w])
}

func lazyBuffer(buf *[]byte, s string, w int, c byte) {
	b := *buf
	if len(b) == 0 {
		if s[w] == c {
			return
		}

		length := len(s)
		if length > cap(b) {
			*buf = make([]byte, length)
		} else {
			*buf = (*buf)[:length]
		}
		b = *buf
		copy(b, s[:w])
	}
	b[w] = c
}
