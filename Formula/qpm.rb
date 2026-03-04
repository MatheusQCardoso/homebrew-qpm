class Qpm < Formula
  desc "Quirino's Package Manager"
  homepage "https://github.com/matheusqcardoso/qpm"
  version "1.0.0"

  if Hardware::CPU.arm?
    url "https://github.com/matheusqcardoso/qpm/releases/download/1.0.0/qpm-1.0.0-darwin-arm64.tar.gz"
    sha256 "ARM_SHA256"
  else
    url "https://github.com/matheusqcardoso/qpm/releases/download/1.0.0/qpm-1.0.0-darwin-amd64.tar.gz"
    sha256 "INTEL_SHA256"
  end

  def install
    bin.install "qpm"
  end

  test do
    system "#{bin}/qpm"
  end
end
