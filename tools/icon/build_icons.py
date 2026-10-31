import os
import shutil
import struct
import sys

import resvg_py

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
ICON = os.path.join(ROOT, "assets", "icon")
MASTER = os.path.join(ICON, "argus-asm.svg")
SMALL = os.path.join(ICON, "argus-asm-16.svg")
PNG_DIR = os.path.join(ICON, "png")
LINUX_DIR = os.path.join(ICON, "linux")
SIZES = [16, 24, 32, 48, 64, 128, 256, 512, 1024]


def render(svg_path, size):
    return bytes(resvg_py.svg_to_bytes(svg_path=svg_path, width=size, height=size))


def png_for(size):
    if size == 16:
        return render(SMALL, 16)
    return render(MASTER, size)


def write_ico(path, sizes):
    images = [(s, png_for(s)) for s in sizes]
    header = struct.pack("<HHH", 0, 1, len(images))
    offset = 6 + 16 * len(images)
    entries = b""
    blobs = b""
    for s, data in images:
        dim = 0 if s >= 256 else s
        entries += struct.pack("<BBBBHHII", dim, dim, 0, 0, 1, 32, len(data), offset + len(blobs))
        blobs += data
    with open(path, "wb") as f:
        f.write(header + entries + blobs)


def write_icns(path):
    chunks = [
        (b"icp4", 16), (b"icp5", 32), (b"icp6", 64), (b"ic07", 128), (b"ic08", 256),
        (b"ic09", 512), (b"ic10", 1024), (b"ic11", 32), (b"ic12", 64), (b"ic13", 256), (b"ic14", 512),
    ]
    body = b""
    for code, px in chunks:
        data = png_for(px)
        body += code + struct.pack(">I", len(data) + 8) + data
    with open(path, "wb") as f:
        f.write(b"icns" + struct.pack(">I", len(body) + 8) + body)


def main():
    os.makedirs(PNG_DIR, exist_ok=True)
    os.makedirs(LINUX_DIR, exist_ok=True)
    for s in SIZES:
        with open(os.path.join(PNG_DIR, "argus-asm-%d.png" % s), "wb") as f:
            f.write(png_for(s))
    write_ico(os.path.join(ICON, "argus-asm.ico"), [16, 24, 32, 48, 64, 128, 256])
    write_icns(os.path.join(ICON, "argus-asm.icns"))
    shutil.copyfile(MASTER, os.path.join(LINUX_DIR, "argus-asm.svg"))
    shutil.copyfile(os.path.join(PNG_DIR, "argus-asm-512.png"), os.path.join(LINUX_DIR, "argus-asm.png"))
    print("ok")


if __name__ == "__main__":
    sys.exit(main())
