package board

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path"
)

type diskDocument struct {
	document
	source string
	info   os.FileInfo
}

func (a *Access) inventory() ([]diskDocument, []IndexFinding, error) {
	var files []diskDocument
	findings := []IndexFinding{}
	for _, scope := range a.policy.scopes() {
		scanned, notes, err := a.scanScope(scope)
		if err != nil {
			return nil, nil, err
		}
		files = append(files, scanned...)
		findings = append(findings, notes...)
	}
	return files, findings, nil
}

func (a *Access) scanScope(scope string) ([]diskDocument, []IndexFinding, error) {
	root, err := a.scopeRoot(scope)
	if os.IsNotExist(err) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, []IndexFinding{{Code: "unsafe_scope", Path: scopePath(scope), Message: err.Error()}}, nil
	}
	defer root.Close()
	entries, err := readDirectory(root)
	if err != nil {
		return nil, nil, err
	}
	var files []diskDocument
	var findings []IndexFinding
	for _, entry := range entries {
		if isTemporary(entry.Name()) {
			continue
		}
		source := path.Join(scopePath(scope), entry.Name())
		doc, valid := documentFromFilename(scope, entry.Name())
		if !valid {
			findings = append(findings, IndexFinding{Code: "unsupported_entry", Path: source})
			continue
		}
		if !entry.Type().IsRegular() {
			findings = append(findings, IndexFinding{Code: "unsafe_entry", Path: source})
			continue
		}
		if doc.ID != "" && (source != doc.canonicalPath() || len(source) > 244) {
			findings = append(findings, IndexFinding{Code: "unsupported_entry", Path: source})
			continue
		}
		digest, info, err := fileDigest(root, entry.Name())
		if err != nil {
			return nil, nil, fmt.Errorf("read %q: %w", source, err)
		}
		doc.Digest = digest
		files = append(files, diskDocument{document: doc, source: source, info: info})
	}
	return files, findings, nil
}

func fileDigest(root *os.Root, name string) (string, os.FileInfo, error) {
	file, err := openRegular(root, name, os.O_RDONLY)
	if err != nil {
		return "", nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", nil, err
	}
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", nil, err
	}
	return fmt.Sprintf("%x", digest.Sum(nil)), info, nil
}
