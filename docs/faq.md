# FAQ

- **Q: Where does my data go? On my HD or somewhere in the cloud?**
- **A**: It is stored locally in a [buku](https://github.com/jarun/buku) compatible sqlite [database](/internal/database/database.go).

----------

- **Q: Does gosuki support bookmark deletion?**
- **A**: No, it only supports adding/modifying bookmarks. Since it's designed to be multi-browser, multi-profile, and real-time, adding deletion functionality would introduce excessive complexity with limited benefit. Users can simply utilize the `#deleted` tag (for example) to organize deleted bookmarks within their own tag hierarchy. Given that bookmarks consume negligible storage space, this approach provides an efficient workaround.

----------

- **Q: Is it possible to synchronize multiple devices?**
- **A**: Yes, you can synchronize your bookmarks using:
  - [P2P Auto Sync](https://gosuki.net/docs/features/multi-device-sync/p2p-auto-sync)
  - [Using Syncthing](https://gosuki.net/docs/features/multi-device-sync/syncthing) 


----------

- **Q: Can I synchronize with my mobile devices?**
- **A**: Yes, using the guides for:
  - **P2P Auto Sync**: [Mobile devices setup](https://gosuki.net/docs/features/multi-device-sync/p2p-auto-sync/#mobile-devices)
  - **Syncthing**: [Example mobile setup](https://gosuki.net/docs/features/multi-device-sync/syncthing/#example)
