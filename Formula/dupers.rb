class Dupers < Formula
  desc "The blazing-fast file duplicate checker and filename search tool"
  homepage "https://github.com/bengarrett/dupers"
  url "https://github.com/bengarrett/dupers/archive/refs/tags/v1.2.1.tar.gz"
  sha256 "fc43baff8da1a03d7ab93465168d3177fe4034dd9ce5a40ddfde75631c6d2624"
  license "LGPL-3.0"
  version "1.2.1"
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