class Qpm < Formula
  desc "Quirino's Package Manager"
  homepage "https://github.com/MatheusQCardoso/homebrew-qpm/"
  version "1.1.0"

  if Hardware::CPU.arm?
    url "https://github.com/MatheusQCardoso/homebrew-qpm/releases/download/1.1.0/qpm-1.1.0-darwin-arm64.tar.gz"
    sha256 "f277ec0ff297dca900653fe933ef548c4dbad541245d426e06ef8f5f734817a5"
  else
    url "https://github.com/MatheusQCardoso/homebrew-qpm/releases/download/1.1.0/qpm-1.1.0-darwin-amd64.tar.gz"
    sha256 "851d10a4e0f1fde108c8711f42d2814837f14a3e332330e49be78ce5c14a7521"
  end

  def install
    bin.install "qpm"
  end

  test do
    system "#{bin}/qpm"
  end
end
