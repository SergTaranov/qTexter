package win

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"syscall"
	"unsafe"
)

const (
	shgfiIcon    = 0x00000100 // SHGFI_ICON
	shgfiBigIcon = 0x00000000 // SHGFI_LARGEICON
	dibRGBColors = 0          // DIB_RGB_COLORS
)

var (
	procSHGetFileInfoW     = shell32.NewProc("SHGetFileInfoW")
	procGetIconInfo        = user32.NewProc("GetIconInfo")
	procDestroyIcon        = user32.NewProc("DestroyIcon")
	procGetDIBits          = gdi32.NewProc("GetDIBits")
	procDeleteObject       = gdi32.NewProc("DeleteObject")
	procCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	procDeleteDC           = gdi32.NewProc("DeleteDC")
	procGetObjectW         = gdi32.NewProc("GetObjectW")
)

type shfileinfo struct { // SHFILEINFOW
	hIcon         uintptr
	iIcon         int32
	dwAttributes  uint32
	szDisplayName [260]uint16
	szTypeName    [80]uint16
}

type iconInfo struct { // ICONINFO
	fIcon    int32
	xHotspot uint32
	yHotspot uint32
	hbmMask  uintptr
	hbmColor uintptr
}

type bitmapHeader struct { // BITMAP
	bmType, bmWidth, bmHeight, bmWidthBytes int32
	bmPlanes, bmBitsPixel                   uint16
	_                                       uint32 // выравнивание перед bmBits
	bmBits                                  uintptr
}

type bitmapInfoHeader struct { // BITMAPINFOHEADER
	biSize          uint32
	biWidth         int32
	biHeight        int32
	biPlanes        uint16
	biBitCount      uint16
	biCompression   uint32
	biSizeImage     uint32
	biXPelsPerMeter int32
	biYPelsPerMeter int32
	biClrUsed       uint32
	biClrImportant  uint32
}

// ExeIconPNG возвращает иконку исполняемого файла или ярлыка в виде
// PNG-байтов (обычно 32×32). Если извлечь не удалось — nil.
func ExeIconPNG(path string) []byte {
	hicon := extractIcon(path)
	if hicon == 0 {
		return nil
	}
	defer procDestroyIcon.Call(hicon)
	img := iconToImage(hicon)
	if img == nil {
		return nil
	}
	var buf bytes.Buffer
	if png.Encode(&buf, img) != nil {
		return nil
	}
	return buf.Bytes()
}

func extractIcon(path string) uintptr {
	ptr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0
	}
	var sfi shfileinfo
	r, _, _ := procSHGetFileInfoW.Call(
		uintptr(unsafe.Pointer(ptr)), 0,
		uintptr(unsafe.Pointer(&sfi)), unsafe.Sizeof(sfi),
		shgfiIcon|shgfiBigIcon)
	if r == 0 {
		return 0
	}
	return sfi.hIcon
}

func iconToImage(hicon uintptr) *image.NRGBA {
	var ii iconInfo
	r, _, _ := procGetIconInfo.Call(hicon, uintptr(unsafe.Pointer(&ii)))
	if r == 0 || ii.hbmColor == 0 {
		return nil
	}
	defer procDeleteObject.Call(ii.hbmColor)
	if ii.hbmMask != 0 {
		defer procDeleteObject.Call(ii.hbmMask)
	}

	var bm bitmapHeader
	r, _, _ = procGetObjectW.Call(ii.hbmColor, unsafe.Sizeof(bm), uintptr(unsafe.Pointer(&bm)))
	if r == 0 || bm.bmWidth <= 0 || bm.bmHeight <= 0 {
		return nil
	}
	w, h := int(bm.bmWidth), int(bm.bmHeight)

	dc, _, _ := procCreateCompatibleDC.Call(0)
	if dc == 0 {
		return nil
	}
	defer procDeleteDC.Call(dc)

	var hdr bitmapInfoHeader
	hdr.biSize = uint32(unsafe.Sizeof(hdr))
	hdr.biWidth = int32(w)
	hdr.biHeight = int32(-h) // top-down, строки сверху вниз
	hdr.biPlanes = 1
	hdr.biBitCount = 32

	buf := make([]byte, w*h*4)
	r, _, _ = procGetDIBits.Call(dc, ii.hbmColor, 0, uintptr(h),
		uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&hdr)), dibRGBColors)
	if r == 0 {
		return nil
	}

	// Старые иконки хранят 32 бита на пиксель без альфы: она вся нулевая.
	// Если хоть один пиксель с ненулевой альфой — считаем альфу настоящей.
	useAlpha := false
	for i := 3; i < len(buf); i += 4 {
		if buf[i] != 0 {
			useAlpha = true
			break
		}
	}

	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := (y*w + x) * 4
			px := buf[i : i+4 : i+4] // порядок в памяти BGRA
			a := px[3]
			if !useAlpha {
				a = 255
			}
			img.SetNRGBA(x, y, color.NRGBA{R: px[2], G: px[1], B: px[0], A: a})
		}
	}
	return img
}
