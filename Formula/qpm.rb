class Qpm < Formula
  desc "Quirino's Package Manager"
  homepage "https://github.com/MatheusQCardoso/homebrew-qpm/"
  version "1.1.0"

  if Hardware::CPU.arm?
    url "https://github.com/MatheusQCardoso/homebrew-qpm/releases/download/1.1.0/qpm-1.1.0-darwin-arm64.tar.gz"
    sha256 "99acd06baf3aa219606d45ca17628f156c01bc4ff95ece3dd52b528315414e22"
  else
    url "https://github.com/MatheusQCardoso/homebrew-qpm/releases/download/1.1.0/qpm-1.1.0-darwin-amd64.tar.gz"
    sha256 "febabf6f71837f514870d8b707d8892a9ea0edffd04d522f541cdac8f7e9b07b"
  end

  def install
    bin.install "qpm"
  end

  test do
    system "#{bin}/qpm"
  end
end
