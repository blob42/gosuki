
# Maintainer: Chakib Benziane (blob42) <contact@blob42.xyz>

pkgname=gosuki-git
_pkgname=${pkgname%-git}
pkgver=v1.3.0
pkgrel=2
pkgdesc="Multi-browser, real-time, extension-free bookmark manager with p2p synchronization and archiving"
arch=('i686' 'x86_64')
makedepends=(git git-lfs go make sqlite)
url='https://github.com/blob42/gosuki'
license=(AGPL3)
sha256sums=(SKIP)
provides=("$_pkgname")
conflicts=("$_pkgname")
source=("git+$url.git")

pkgver() {
  cd ${pkgname%-git}
  git describe --tags | sed "s+-+.r+" | tr - .

}

prepare() {
    msg2 'Setting up repository...'
    cd "${_pkgname}"
    git lfs install --local
    if ! git remote -v | grep -q 'network-origin'; then
      git remote add network-origin ${url};
    fi
    git lfs fetch network-origin
    git lfs checkout
}

build() {
  cd "${_pkgname}"

  msg2 'Building...'
  make SYSTRAY=true release

  msg2 'Generating shell completions...'
  for _sh in bash zsh fish; do
    build/gosuki -S completion $_sh > contrib/$_sh.completions
  done
}

package() {
  cd ${_pkgname}


  install -Dm 755 build/gosuki -t "$pkgdir"/usr/bin
  install -Dm 755 build/suki -t "$pkgdir"/usr/bin

  install -Dm 644 contrib/linux/etc/systemd/user/${_pkgname}.service \
    "$pkgdir/usr/lib/systemd/user/${_pkgname}.service"


  # completions
  install -Dm 644 contrib/fish.completions \
    "${pkgdir}/usr/share/fish/completions/${_pkgname}.fish"

  install -Dm 644 contrib/bash.completions \
    "${pkgdir}/usr/share/bash-completion/completions/${_pkgname}"

  install -Dm 644 contrib/zsh.completions \
    "${pkgdir}/usr/share/zsh/site-functions/_${_pkgname}"

  # documentation
  for _doc in README.md; do
    install -Dm 644 $_doc -t "$pkgdir"/usr/share/doc/"${_pkgname}"
  done

  install -Dm 755 contrib/rofi.sh "${pkgdir}/usr/share/${_pkgname}/scripts/rofi-suki.sh"


  # marktab
  install -Dm 644 contrib/marktab/example.marktab "${pkgdir}/usr/share/doc/${_pkgname}/example.marktab"

  for mt_script in $(find contrib/marktab/scripts -type f -print); do
    msg2 $mt_script
    install -Dm 755 $mt_script "${pkgdir}/usr/share/${_pkgname}/scripts/marktab/$(basename $mt_script)"
  done

  install -Dm 644 contrib/${_pkgname}.1 \
    "${pkgdir}/usr/share/man/man1/${_pkgname}.1"

  install -Dm 644 contrib/suki.1 -t \
    "${pkgdir}/usr/share/man/man1/"


  # cleaning up pkgdir"
  find "$pkgdir" -type d -name .git -exec rm -r '{}' + 2>/dev/null || true
}
