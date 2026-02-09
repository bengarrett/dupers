class Dupers < Formula
  desc "The blazing-fast file duplicate checker and filename search tool"
  homepage "https://github.com/bengarrett/dupers"
  url "https://github.com/bengarrett/dupers/archive/refs/tags/v1.1.3.tar.gz"
  sha256 "f261ec5228f112e1d025c1b5fdba79b86e68e4d14ddf45915ab7ac636f244446"
  license "LGPL-3.0"
  version "1.1.3"
  head "https://github.com/bengarrett/dupers.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "."
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