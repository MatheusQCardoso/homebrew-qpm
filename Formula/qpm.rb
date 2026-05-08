class Qpm < Formula
  desc "Quirino's Package Manager"
  homepage "https://github.com/MatheusQCardoso/homebrew-qpm/"
  version "1.1.0"

  if Hardware::CPU.arm?
    url "https://github.com/MatheusQCardoso/homebrew-qpm/releases/download/1.1.0/qpm-1.1.0-darwin-arm64.tar.gz"
    sha256 "a2eccfb9768647497bad33a08fe19a9d3ccf8e5b29a3f216507546f115bc81a5"
  else
    url "https://github.com/MatheusQCardoso/homebrew-qpm/releases/download/1.1.0/qpm-1.1.0-darwin-amd64.tar.gz"
    sha256 "a55350037466d2938c022200934b9c0ce43bc083052acfc541b7e1d31a73729f"
  end

  def install
    bin.install "qpm"
  end

  test do
    system "#{bin}/qpm"
  end
end
