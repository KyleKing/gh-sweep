# typed: false
# frozen_string_literal: true

class GhSweep < Formula
  desc "Bubble Tea TUI gh extension for cross-repository GitHub maintenance"
  homepage "https://github.com/KyleKing/gh-sweep"
  license "MIT"
  version "0.1.0"

  on_macos do
    if Hardware::CPU.arm?
      url "#{homepage}/releases/download/v#{version}/gh-sweep-darwin-arm64"
      sha256 "REPLACE_WITH_SHA256_FOR_DARWIN_ARM64"
    else
      url "#{homepage}/releases/download/v#{version}/gh-sweep-darwin-amd64"
      sha256 "REPLACE_WITH_SHA256_FOR_DARWIN_AMD64"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "#{homepage}/releases/download/v#{version}/gh-sweep-linux-arm64"
      sha256 "REPLACE_WITH_SHA256_FOR_LINUX_ARM64"
    else
      url "#{homepage}/releases/download/v#{version}/gh-sweep-linux-amd64"
      sha256 "REPLACE_WITH_SHA256_FOR_LINUX_AMD64"
    end
  end

  def install
    binary_name = "gh-sweep-#{OS.kernel_name.downcase}-#{Hardware::CPU.arch}"
    bin.install binary_name => "gh-sweep"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/gh-sweep --version")
  end
end
