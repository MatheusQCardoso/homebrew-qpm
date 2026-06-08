version_file = File.expand_path("../VERSION", __dir__)
VERSION_STRING = if File.exist?(version_file)
  File.read(version_file).strip
else
  raise "VERSION file not found at #{version_file}"
end

class Qpm < Formula
  desc "Quirino's Package Manager"
  homepage "https://github.com/MatheusQCardoso/homebrew-qpm/"
  version VERSION_STRING

  if Hardware::CPU.arm?
    url "https://github.com/MatheusQCardoso/homebrew-qpm/releases/download/#{VERSION_STRING}/qpm-#{VERSION_STRING}-darwin-arm64.tar.gz"
    sha256 "35f54e177301f81ed67f501df0d646634966b8fe0f258d1c8a38229fd4a5d7f6"
  else
    url "https://github.com/MatheusQCardoso/homebrew-qpm/releases/download/#{VERSION_STRING}/qpm-#{VERSION_STRING}-darwin-amd64.tar.gz"
    sha256 "74b9d9f23e732c4a3bae836fa078ef29aa3d2a149813946dfff6a0c98143ef0d"
  end

  def install
    bin.install "qpm"
  end

  test do
    system "#{bin}/qpm"
  end
end
