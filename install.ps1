# Installs quiver on Windows from its GitHub release.
#
#   irm https://raw.githubusercontent.com/kheif/quiver/main/install.ps1 | iex
#
# QUIVER_VERSION      release to install, such as v1.0.0 (default: the latest)
# QUIVER_INSTALL_DIR  where to put quiver.exe (default: %LOCALAPPDATA%\Programs\quiver)
# QUIVER_DOWNLOAD_URL folder with the archives and checksums.txt, for mirrors
#                     and tests: a URL, a file:// URL or a local folder
#                     (default: the GitHub release)
#
# The folder is added to the PATH of the user, so no admin rights are needed.

# iex runs this in the caller's shell: everything stays inside the script
# block, and errors throw instead of exit, which would close the shell.
& {
	$ErrorActionPreference = 'Stop'
	# The progress bar makes downloads much slower in Windows PowerShell 5.1.
	$ProgressPreference = 'SilentlyContinue'

	$repo = 'kheif/quiver'
	$version = if ($env:QUIVER_VERSION) { $env:QUIVER_VERSION } else { 'latest' }
	$dir = if ($env:QUIVER_INSTALL_DIR) { $env:QUIVER_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'Programs\quiver' }

	function Fail([string]$msg) {
		throw "quiver install: $msg"
	}

	if ([Environment]::OSVersion.Platform -ne 'Win32NT') {
		Fail 'install.ps1 is for Windows; on macOS and Linux, install with: curl -fsSL https://raw.githubusercontent.com/kheif/quiver/main/install.sh | sh'
	}
	# A 32-bit PowerShell on 64-bit Windows sees x86; PROCESSOR_ARCHITEW6432
	# has the real one.
	$cpu = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
	$arch = switch ($cpu) {
		'AMD64' { 'amd64' }
		'ARM64' { 'arm64' }
		default { Fail "unsupported CPU $cpu" }
	}

	if ($env:QUIVER_DOWNLOAD_URL) {
		$base = $env:QUIVER_DOWNLOAD_URL.TrimEnd('/', '\')
	} elseif ($version -eq 'latest') {
		$base = "https://github.com/$repo/releases/latest/download"
	} else {
		$base = "https://github.com/$repo/releases/download/$version"
	}
	$archive = "quiver_windows_$arch.zip"

	# Windows PowerShell 5.1 may not offer TLS 1.2, which GitHub requires.
	[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

	# Save-Download saves the file name from base to path. A local folder or
	# file:// URL is copied, since Invoke-WebRequest does not read files.
	function Save-Download([string]$name, [string]$path) {
		$uri = $null
		if ([Uri]::TryCreate($base, [UriKind]::Absolute, [ref]$uri) -and -not $uri.IsFile) {
			try {
				Invoke-WebRequest -UseBasicParsing -Uri "$base/$name" -OutFile $path
			} catch {
				Fail "could not download $base/${name}: $($_.Exception.Message)"
			}
			return
		}
		$from = if ($uri) { Join-Path $uri.LocalPath $name } else { Join-Path $base $name }
		if (-not (Test-Path -LiteralPath $from -PathType Leaf)) {
			Fail "could not download ${from}: not found"
		}
		Copy-Item -LiteralPath $from -Destination $path
	}

	$tmp = Join-Path ([IO.Path]::GetTempPath()) ("quiver-install-" + [Guid]::NewGuid())
	New-Item -ItemType Directory -Path $tmp | Out-Null
	try {
		Write-Host "Downloading $archive ($version)..."
		Save-Download $archive (Join-Path $tmp $archive)
		Save-Download 'checksums.txt' (Join-Path $tmp 'checksums.txt')

		$want = $null
		foreach ($line in Get-Content -LiteralPath (Join-Path $tmp 'checksums.txt')) {
			$fields = -split $line
			if ($fields.Count -eq 2 -and $fields[1] -eq $archive) {
				$want = $fields[0]
			}
		}
		if (-not $want) {
			Fail "$archive is not in checksums.txt"
		}
		$got = (Get-FileHash -Algorithm SHA256 -LiteralPath (Join-Path $tmp $archive)).Hash
		if ($got -ne $want) {
			Fail "checksum mismatch for $archive; the download is broken or was changed"
		}

		Expand-Archive -LiteralPath (Join-Path $tmp $archive) -DestinationPath (Join-Path $tmp 'x')
		New-Item -ItemType Directory -Force -Path $dir | Out-Null
		$exe = Join-Path $dir 'quiver.exe'
		# A running quiver.exe, such as an assistant's MCP server, cannot be
		# replaced, but it can be renamed; it keeps running from the old file.
		$old = "$exe.old"
		if (Test-Path -LiteralPath $old) {
			Remove-Item -LiteralPath $old -Force -ErrorAction SilentlyContinue
		}
		if (Test-Path -LiteralPath $exe) {
			if (Test-Path -LiteralPath $old) {
				Fail "$old is still in use; close the assistants that run quiver and try again"
			}
			Move-Item -LiteralPath $exe -Destination $old
		}
		Move-Item -LiteralPath (Join-Path $tmp 'x\quiver.exe') -Destination $exe
	} finally {
		Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
	}

	Write-Host "Installed $(& $exe --version) to $exe"

	# Add the folder to the user's PATH in the registry. The raw value keeps
	# entries such as %USERPROFILE%\bin, which [Environment] would expand.
	$key = Get-Item -Path 'HKCU:\Environment'
	$userPath = $key.GetValue('Path', '', 'DoNotExpandEnvironmentNames')
	$entries = @($userPath -split ';' | Where-Object { $_ })
	$full = [IO.Path]::GetFullPath($dir).TrimEnd('\')
	$onPath = $entries | Where-Object { [Environment]::ExpandEnvironmentVariables($_).TrimEnd('\') -eq $full }
	if (-not $onPath) {
		$kind = if ($userPath) { $key.GetValueKind('Path') } else { [Microsoft.Win32.RegistryValueKind]::ExpandString }
		$value = (@($entries) + $full) -join ';'
		Set-ItemProperty -Path 'HKCU:\Environment' -Name 'Path' -Value $value -Type $kind
		# Setting a variable through [Environment] tells running programs,
		# such as Explorer, that the environment changed, so new shells get
		# the new PATH.
		[Environment]::SetEnvironmentVariable('QUIVER_INSTALL_PATH_CHANGED', '1', 'User')
		[Environment]::SetEnvironmentVariable('QUIVER_INSTALL_PATH_CHANGED', $null, 'User')
		$env:Path = "$env:Path;$full"
		Write-Host ''
		Write-Host "Added $full to your PATH. Open a new terminal to use quiver there."
	}
	Write-Host ''
	Write-Host 'Next: quiver mcp install'
}
