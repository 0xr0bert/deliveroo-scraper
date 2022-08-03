def clean_pounds(price):
    return float(price.replace(",", "")[1:])
