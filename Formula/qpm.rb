version = File.read(File.expand_path("../VERSION", __dir__)).strip

class Qpm < Formula
  desc "Quirino's Package Manager"
  homepage "https://github.com/MatheusQCardoso/homebrew-qpm/"
  version version

  if Hardware::CPU.arm?
    url "https://github.com/MatheusQCardoso/homebrew-qpm/releases/download/#{version}/qpm-#{version}-darwin-arm64.tar.gz"
    sha256 "f924ad592fd122043117ec1d4b73d3b24c3553d04197aa5d610e9620e55cec91"
  else
    url "https://github.com/MatheusQCardoso/homebrew-qpm/releases/download/#{version}/qpm-#{version}-darwin-amd64.tar.gz"
    sha256 "d0990f61cf1d9d1347ac2968210c40ab46cf59a03db4cc2923a8ae82f81f7d2d"
  end

  def install
    bin.install "qpm"
  end

  test do
    system "#{bin}/qpm"
  end
end
