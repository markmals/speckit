# Canonical source for the markmals/homebrew-tap formula. On the first release
# this file (and update-specify.yml) is copied into markmals/homebrew-tap; after
# that the tap is the source of truth and update-specify.yml self-bumps the
# version + sha256 on each new SpecKit release. See README.md in this directory.
#
# From-source, bottled by the tap's brew test-bot CI — `brew install
# markmals/tap/specify` installs the prebuilt bottle, not a goreleaser download.
class Specify < Formula
  desc "Spec-driven development engine for native, multiplatform apps"
  homepage "https://github.com/markmals/speckit"
  url "https://github.com/markmals/speckit/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "0000000000000000000000000000000000000000000000000000000000000000" # set on first release
  license "MIT"
  head "https://github.com/markmals/speckit.git", branch: "main"

  depends_on "go" => :build

  def install
    ldflags = "-s -w -X main.version=#{version}"
    system "go", "build", *std_go_args(ldflags: ldflags), "./cmd/specify"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/specify version")
  end
end
