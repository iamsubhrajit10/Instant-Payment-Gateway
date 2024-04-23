wrk.method = 'POST'
wrk.headers['Content-Type'] = 'application/json'
counter = 0
request = function()
  counter = counter + 1
  local paymentId1 = (counter * 2) - 1
  local paymentId2 = counter * 2
  local body = [[
{
    "Requests":[
        {
            "TransactionID": "]] .. counter .. [[",
            "PaymentID": "]] .. paymentId1 .. [[",
            "Type": "resolve"
        },
        {
            "TransactionID": "]] .. counter .. [[",
            "PaymentID": "]] .. paymentId2 .. [[",
            "Type": "resolve"
        }
    ]
}
]]
  return wrk.format(nil, nil, nil, body)
end
