# E2E test fixtures

Small, committed sample files used by e2e tests that need real binary
input rather than a workflow-generated one.

- `ocr-sample.png` — a 400x100 PNG reading "KDeps OCR Sample", used by
  `test_features_ocr.sh`'s end-to-end test. Regenerate with:
  ```bash
  magick -size 400x100 xc:white -gravity center -pointsize 28 -fill black \
    -annotate 0 "KDeps OCR Sample" ocr-sample.png
  ```
- `transcribe-sample.mp3` — a ~2s speech clip saying "This is a kdeps
  transcription test.", used by `test_features_transcribe.sh`'s
  API-key-gated end-to-end test. Regenerate with:
  ```bash
  say -o transcribe-sample.aiff "This is a kdeps transcription test."
  ffmpeg -y -i transcribe-sample.aiff -codec:a libmp3lame -qscale:a 4 transcribe-sample.mp3
  rm transcribe-sample.aiff
  ```
