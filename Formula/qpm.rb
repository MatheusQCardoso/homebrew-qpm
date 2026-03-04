class Qpm < Formula
  desc "Quirino's Package Manager"
  homepage "https://github.com/matheusqcardoso/qpm"
  version "1.0.0"

  if Hardware::CPU.arm?
    url "https://github.com/MatheusQCardoso/qpm/releases/download/1.0.0/qpm-1.0.0-darwin-arm64.tar.gz"
    sha256 "d3e5dd3f375804c355d458665696e865ba2248667d5be847c63f8bfafe3b5fe2"
  else
    url "https://github.com/MatheusQCardoso/qpm/releases/download/1.0.0/qpm-1.0.0-darwin-amd64.tar.gz"
    sha256 "eaac5dd0208a54f125e6c3cd8bc0b4c96ee50f3983e3809a77cf79c5b2b3117e"
  end

  def install
    bin.install "qpm"
  end

  test do
    system "#{bin}/qpm"
  end
end
