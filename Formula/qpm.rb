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
    sha256 "d0ddef8eadff356869eb9be2fbb02b875882caa96326d63ed3ea86ba50a0c477"
  else
    url "https://github.com/MatheusQCardoso/homebrew-qpm/releases/download/#{VERSION_STRING}/qpm-#{VERSION_STRING}-darwin-amd64.tar.gz"
    sha256 "d068fc5dffb6b87251bd6595bb8dddc6ebdb814bae6a58ced3aba082f9253eb1"
  end

  def install
    bin.install "qpm"
  end

  test do
    system "#{bin}/qpm"
  end
end
