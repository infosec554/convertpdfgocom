package gotenberg

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

type Client interface {
	PDFToWord(ctx context.Context, pdfPath string) ([]byte, error)
	WordToPDF(ctx context.Context, wordPath string) ([]byte, error)
	ExcelToPDF(ctx context.Context, excelPath string) ([]byte, error)
	PowerPointToPDF(ctx context.Context, pptPath string) ([]byte, error)
	HTMLToPDF(ctx context.Context, htmlPath string) ([]byte, error)
}

type gotenbergClient struct {
	baseURL string
}

func New(url string) Client {
	return &gotenbergClient{baseURL: url}
}

// PDF -> Word
func (g *gotenbergClient) PDFToWord(ctx context.Context, pdfPath string) ([]byte, error) {
	file, err := os.Open(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %w", err)
	}
	defer file.Close()

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	part, err := writer.CreateFormFile("files", filepath.Base(pdfPath))
	if err != nil {
		return nil, fmt.Errorf("cannot create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("cannot copy file: %w", err)
	}

	_ = writer.WriteField("output", "docx") // optional
	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", g.baseURL+"/forms/libreoffice/convert", &requestBody)
	if err != nil {
		return nil, fmt.Errorf("cannot create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("conversion failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("conversion failed: %s", string(bodyBytes))
	}

	return io.ReadAll(resp.Body)
}

// Word -> PDF
func (g *gotenbergClient) WordToPDF(ctx context.Context, wordPath string) ([]byte, error) {
	file, err := os.Open(wordPath)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %w", err)
	}
	defer file.Close()

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	part, err := writer.CreateFormFile("files", filepath.Base(wordPath))
	if err != nil {
		return nil, fmt.Errorf("cannot create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("cannot copy file: %w", err)
	}

	_ = writer.WriteField("waitTimeout", "30s")
	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", g.baseURL+"/forms/libreoffice/convert", &requestBody)
	if err != nil {
		return nil, fmt.Errorf("cannot create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("conversion failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("conversion failed: %s", string(bodyBytes))
	}

	return io.ReadAll(resp.Body)
}

// Excel -> PDF
func (g *gotenbergClient) ExcelToPDF(ctx context.Context, excelPath string) ([]byte, error) {
	file, err := os.Open(excelPath)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %w", err)
	}
	defer file.Close()

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	part, err := writer.CreateFormFile("files", filepath.Base(excelPath))
	if err != nil {
		return nil, fmt.Errorf("cannot create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("cannot copy file: %w", err)
	}

	_ = writer.WriteField("waitTimeout", "30s")
	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", g.baseURL+"/forms/libreoffice/convert", &requestBody)
	if err != nil {
		return nil, fmt.Errorf("cannot create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("conversion failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("conversion failed: %s", string(bodyBytes))
	}

	return io.ReadAll(resp.Body)
}

// PowerPoint -> PDF
func (g *gotenbergClient) PowerPointToPDF(ctx context.Context, pptPath string) ([]byte, error) {
	file, err := os.Open(pptPath)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %w", err)
	}
	defer file.Close()

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	part, err := writer.CreateFormFile("files", filepath.Base(pptPath))
	if err != nil {
		return nil, fmt.Errorf("cannot create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("cannot copy file: %w", err)
	}

	_ = writer.WriteField("waitTimeout", "30s") // optional
	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", g.baseURL+"/forms/libreoffice/convert", &requestBody)
	if err != nil {
		return nil, fmt.Errorf("cannot create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("conversion failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("conversion failed: %s", string(bodyBytes))
	}

	return io.ReadAll(resp.Body)
}

func (g *gotenbergClient) HTMLToPDF(ctx context.Context, htmlPath string) ([]byte, error) {
	file, err := os.Open(htmlPath)
	if err != nil {
		return nil, fmt.Errorf("cannot open HTML file: %w", err)
	}
	defer file.Close()

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Fayl nomi **faqat** "index.html" bo‘lishi kerak
	part, err := writer.CreateFormFile("files", "index.html")
	if err != nil {
		return nil, fmt.Errorf("cannot create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("cannot copy HTML file: %w", err)
	}

	writer.Close()

	// ✅ Gotenberg 7 endpoint
	req, err := http.NewRequestWithContext(ctx, "POST", g.baseURL+"/forms/html/convert", &requestBody)
	if err != nil {
		return nil, fmt.Errorf("cannot create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("conversion request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gotenberg error: %s", string(bodyBytes))
	}

	return io.ReadAll(resp.Body)
}
