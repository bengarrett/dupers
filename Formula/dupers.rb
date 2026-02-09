class Dupers < Formula
  desc "The blazing-fast file duplicate checker and filename search tool"
  homepage "https://github.com/bengarrett/dupers"
  url "https://github.com/bengarrett/dupers/archive/refs/tags/v1.2.0.tar.gz"
  sha256 "f961d5206f6c6839618028ee318895ab336cdc0609edf20ec41f7eb6c34b120e"
  license "LGPL-3.0"
  version "1.2.0"
  head "https://github.com/bengarrett/dupers.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w -X main.version=#{version}"), "."
  end

  test do
    # Create a test file
    test_file = testpath/"test.txt"
    test_file.write "test content"
    
    # Test that dupers can run
    system bin/"dupers", "-help"
    
    # Test basic functionality
    system bin/"dupers", "search", testpath
  end
end