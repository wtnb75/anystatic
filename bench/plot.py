import matplotlib.pyplot as plt
import argparse
from pathlib import Path


def main():
    parser = argparse.ArgumentParser("plot")
    parser.add_argument("files", nargs="*")
    options = parser.parse_args()
    data = {}
    for i in options.files:
        data[i] = {}
        data[i]["x"] = []
        data[i]["y"] = []
        with open(i) as ifp:
            d1 = [[float(x) for x in line.split()] for line in ifp.readlines()]
            for d in d1:
                data[i]["x"].append(d[0])
                data[i]["y"].append(d[1])
    fig, ax = plt.subplots()
    replacemap = {
        "bencht": "traefik plugin",
        "bencha": "anystatic standalone",
        "benchn": "nginx",
        ".small.txt.": ", small, ",
        ".medium.txt.": ", medium, ",
        ".large.txt.": ", large, ",
    }
    for i in options.files:
        lb = Path(i).name.removeprefix("summary.")
        for k, v in replacemap.items():
            lb = lb.replace(k, v)
        ax.plot(data[i]["x"], data[i]["y"], ".-", label=lb)
    ax.legend(loc="upper left")
    ax.set_xlabel("req/seq")
    ax.set_ylabel("latency(us), 99%tile")
    plt.show()


if __name__ == "__main__":
    main()
